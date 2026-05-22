package main

import "nearfy/internal/model"

func (a *App) GetDevices() []map[string]interface{} {
	if a.discovery == nil {
		return nil
	}
	devices := a.discovery.GetDevices()
	result := make([]map[string]interface{}, 0, len(devices))
	for _, d := range devices {
		result = append(result, map[string]interface{}{
			"node_id": d.NodeID,
			"name":    d.Name,
			"ip":      d.IP,
			"port":    d.Port,
			"os":      d.OS,
			"online":  d.Online,
		})
	}
	return result
}

func (a *App) GetLocalInfo() map[string]interface{} {
	if a.discovery == nil {
		return nil
	}
	return map[string]interface{}{
		"node_id": a.discovery.NodeID(),
		"name":    model.GetHostname(),
		"os":      model.GetOSName(),
	}
}
