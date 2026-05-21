import { useState, useEffect } from 'react';

export interface Device {
  node_id: string;
  name: string;
  ip: string;
  port: number;
  os: string;
  online: boolean;
}

export function useDevices() {
  const [devices, setDevices] = useState<Device[]>([]);
  const [localInfo, setLocalInfo] = useState<{ node_id: string; name: string; os: string } | null>(null);

  useEffect(() => {
    // Import Wails bindings dynamically
    import('../../wailsjs/go/main/App').then(({ GetLocalInfo, GetDevices }) => {
      GetLocalInfo().then((info) => setLocalInfo(info as { node_id: string; name: string; os: string }));
      GetDevices().then((devs) => setDevices(devs as Device[]));
    });

    import('../../wailsjs/runtime/runtime').then(({ EventsOn }) => {
      EventsOn('device:changed', (data: Device) => {
        setDevices(prev => {
          const idx = prev.findIndex(d => d.node_id === data.node_id);
          if (data.online) {
            if (idx >= 0) {
              const next = [...prev];
              next[idx] = data;
              return next;
            }
            return [...prev, data];
          }
          return prev.filter(d => d.node_id !== data.node_id);
        });
      });
    });
  }, []);

  return { devices, localInfo };
}
