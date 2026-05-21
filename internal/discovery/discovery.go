package discovery

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"runtime"
	"sync"
	"time"

	"github.com/google/uuid"
)

const (
	DefaultPort      = 19876
	MaxPortAttempts  = 5
	BroadcastInterval = 3 * time.Second
	OnlineTTL        = 10 * time.Second
	cleanupInterval  = 2 * time.Second
	udpMaxSize       = 4096
)

// BroadcastMessage is the JSON payload sent over UDP.
type BroadcastMessage struct {
	NodeID    string `json:"node_id"`
	Name      string `json:"name"`
	IP        string `json:"ip"`
	Port      int    `json:"port"`
	OS        string `json:"os"`
	Timestamp int64  `json:"timestamp"`
	Leave     bool   `json:"leave,omitempty"`
}

// DeviceEntry represents a discovered peer.
type DeviceEntry struct {
	NodeID   string
	Name     string
	IP       string
	Port     int
	OS       string
	Online   bool
	LastSeen time.Time
}

// DeviceChangeCallback is called when a device goes online or offline.
type DeviceChangeCallback func(device *DeviceEntry, online bool)

// Service handles UDP broadcast discovery on the LAN.
type Service struct {
	nodeID   string
	nodeName string
	tcpPort  int
	os       string

	mu       sync.RWMutex
	devices  map[string]*DeviceEntry
	callback DeviceChangeCallback

	stopCh   chan struct{}
	conn     *net.UDPConn
	port     int
}

// NewService creates a new discovery service. nodeName is the display name,
// tcpPort is the port the file-transfer TCP server listens on.
func NewService(nodeName string, tcpPort int) *Service {
	return &Service{
		nodeID:   uuid.New().String(),
		nodeName: nodeName,
		tcpPort:  tcpPort,
		os:       runtime.GOOS,
		devices:  make(map[string]*DeviceEntry),
		stopCh:   make(chan struct{}),
	}
}

// SetCallback registers a callback for device online/offline events.
func (s *Service) SetCallback(cb DeviceChangeCallback) {
	s.callback = cb
}

// NodeID returns the unique node identifier of this service instance.
func (s *Service) NodeID() string {
	return s.nodeID
}

// SetTCPPort updates the TCP port advertised in broadcast messages.
func (s *Service) SetTCPPort(port int) {
	s.tcpPort = port
}

// Start binds the UDP port and starts the listener, broadcaster, and cleanup goroutines.
func (s *Service) Start() error {
	conn, port, err := s.bindPort()
	if err != nil {
		return fmt.Errorf("discovery: failed to bind UDP port: %w", err)
	}
	s.conn = conn
	s.port = port

	go s.listen(conn)
	go s.broadcastLoop()
	go s.cleanupLoop()

	return nil
}

// Stop signals all goroutines to stop and sends a leave message.
func (s *Service) Stop() {
	close(s.stopCh)
	s.sendLeave()
	if s.conn != nil {
		s.conn.Close()
	}
}

// GetDevices returns a thread-safe copy of the current device list.
func (s *Service) GetDevices() []*DeviceEntry {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*DeviceEntry, 0, len(s.devices))
	for _, d := range s.devices {
		result = append(result, &DeviceEntry{
			NodeID:   d.NodeID,
			Name:     d.Name,
			IP:       d.IP,
			Port:     d.Port,
			OS:       d.OS,
			Online:   d.Online,
			LastSeen: d.LastSeen,
		})
	}
	return result
}

// --- internal methods ---

// bindPort tries to bind a UDP port starting from DefaultPort up to MaxPortAttempts.
func (s *Service) bindPort() (*net.UDPConn, int, error) {
	for i := 0; i < MaxPortAttempts; i++ {
		port := DefaultPort + i
		addr, err := net.ResolveUDPAddr("udp4", fmt.Sprintf(":%d", port))
		if err != nil {
			continue
		}
		conn, err := net.ListenUDP("udp4", addr)
		if err == nil {
			return conn, port, nil
		}
	}
	return nil, 0, fmt.Errorf("no available port in range %d-%d", DefaultPort, DefaultPort+MaxPortAttempts-1)
}

// listen reads UDP packets and processes them.
func (s *Service) listen(conn *net.UDPConn) {
	buf := make([]byte, udpMaxSize)
	for {
		select {
		case <-s.stopCh:
			return
		default:
		}

		conn.SetReadDeadline(time.Now().Add(1 * time.Second))
		n, _, err := conn.ReadFromUDP(buf)
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				continue
			}
			select {
			case <-s.stopCh:
				return
			default:
				continue
			}
		}

		var msg BroadcastMessage
		if err := json.Unmarshal(buf[:n], &msg); err != nil {
			continue
		}

		// Ignore own messages
		if msg.NodeID == s.nodeID {
			continue
		}

		if msg.Leave {
			s.removeDevice(msg.NodeID)
		} else {
			s.updateDevice(msg)
		}
	}
}

// broadcastLoop sends broadcast messages at regular intervals.
func (s *Service) broadcastLoop() {
	ticker := time.NewTicker(BroadcastInterval)
	defer ticker.Stop()

	// Send an initial broadcast immediately
	s.broadcast()

	for {
		select {
		case <-s.stopCh:
			return
		case <-ticker.C:
			s.broadcast()
		}
	}
}

// cleanupLoop periodically removes stale devices.
func (s *Service) cleanupLoop() {
	ticker := time.NewTicker(cleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-s.stopCh:
			return
		case <-ticker.C:
			s.removeStale()
		}
	}
}

// updateDevice adds or updates a device entry and fires the callback.
func (s *Service) updateDevice(msg BroadcastMessage) {
	s.mu.Lock()
	defer s.mu.Unlock()

	entry, exists := s.devices[msg.NodeID]
	if exists {
		entry.Name = msg.Name
		entry.IP = msg.IP
		entry.Port = msg.Port
		entry.OS = msg.OS
		entry.Online = true
		entry.LastSeen = time.Unix(msg.Timestamp, 0)
	} else {
		entry = &DeviceEntry{
			NodeID:   msg.NodeID,
			Name:     msg.Name,
			IP:       msg.IP,
			Port:     msg.Port,
			OS:       msg.OS,
			Online:   true,
			LastSeen: time.Unix(msg.Timestamp, 0),
		}
		s.devices[msg.NodeID] = entry
	}

	if s.callback != nil {
		cb := s.callback
		go cb(entry, true)
	}
}

// removeDevice removes a device by nodeID and fires the callback.
func (s *Service) removeDevice(nodeID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	entry, exists := s.devices[nodeID]
	if !exists {
		return
	}
	delete(s.devices, nodeID)

	if s.callback != nil {
		cb := s.callback
		entry.Online = false
		go cb(entry, false)
	}
}

// removeStale removes devices whose LastSeen is older than OnlineTTL.
func (s *Service) removeStale() {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	for id, entry := range s.devices {
		if now.Sub(entry.LastSeen) > OnlineTTL {
			delete(s.devices, id)
			if s.callback != nil {
				cb := s.callback
				entry.Online = false
				go cb(entry, false)
			}
		}
	}
}

// broadcast sends a BroadcastMessage to all broadcast addresses.
func (s *Service) broadcast() {
	ips := getPrivateIPs()
	if len(ips) == 0 {
		return
	}

	msg := BroadcastMessage{
		NodeID:    s.nodeID,
		Name:      s.nodeName,
		Port:      s.tcpPort,
		OS:        s.os,
		Timestamp: time.Now().Unix(),
	}

	data, err := json.Marshal(msg)
	if err != nil {
		log.Printf("discovery: marshal error: %v", err)
		return
	}

	for _, ip := range ips {
		bcast := getBroadcastIP(ip)
		addr, err := net.ResolveUDPAddr("udp4", fmt.Sprintf("%s:%d", bcast, s.port))
		if err != nil {
			continue
		}
		if s.conn != nil {
			s.conn.WriteToUDP(data, addr)
		}
	}
}

// sendLeave sends a leave BroadcastMessage to all broadcast addresses.
func (s *Service) sendLeave() {
	ips := getPrivateIPs()
	if len(ips) == 0 {
		return
	}

	msg := BroadcastMessage{
		NodeID:    s.nodeID,
		Name:      s.nodeName,
		OS:        s.os,
		Timestamp: time.Now().Unix(),
		Leave:     true,
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return
	}

	for _, ip := range ips {
		bcast := getBroadcastIP(ip)
		addr, err := net.ResolveUDPAddr("udp4", fmt.Sprintf("%s:%d", bcast, s.port))
		if err != nil {
			continue
		}
		if s.conn != nil {
			s.conn.WriteToUDP(data, addr)
		}
	}
}

// getPrivateIPs returns all private IPv4 addresses on this host.
func getPrivateIPs() []string {
	var ips []string

	ifaces, err := net.Interfaces()
	if err != nil {
		return ips
	}

	for _, iface := range ifaces {
		// Skip loopback and down interfaces
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addrs {
			ipNet, ok := addr.(*net.IPNet)
			if !ok {
				continue
			}
			ip := ipNet.IP
			if ip.IsLoopback() || ip.To4() == nil {
				continue
			}
			if isPrivateIPv4(ip) {
				ips = append(ips, ip.String())
			}
		}
	}

	return ips
}

// isPrivateIPv4 checks if an IP is a private IPv4 address (10.x, 172.16-31.x, 192.168.x).
func isPrivateIPv4(ip net.IP) bool {
	ip = ip.To4()
	if ip == nil {
		return false
	}
	return ip[0] == 10 ||
		(ip[0] == 172 && ip[1] >= 16 && ip[1] <= 31) ||
		(ip[0] == 192 && ip[1] == 168)
}

// getBroadcastIP converts a private IPv4 address to its broadcast address (x.x.x.255).
func getBroadcastIP(ip string) string {
	parsed := net.ParseIP(ip).To4()
	if parsed == nil {
		return ip
	}
	bcast := make(net.IP, len(parsed))
	copy(bcast, parsed)
	bcast[3] = 255
	return bcast.String()
}
