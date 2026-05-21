import { useState, useEffect } from 'react';
import { Events } from '@wailsio/runtime';
import { GetLocalInfo, GetDevices } from '../../bindings/lan-file-share/app.js';

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
    GetLocalInfo().then((info: any) => setLocalInfo(info));
    GetDevices().then((devs: any) => setDevices(devs));

    Events.On('device:changed', (ev: any) => {
      const d = ev.data as Device;
      setDevices(prev => {
        const idx = prev.findIndex(dev => dev.node_id === d.node_id);
        if (d.online) {
          if (idx >= 0) {
            const next = [...prev];
            next[idx] = d;
            return next;
          }
          return [...prev, d];
        }
        return prev.filter(dev => dev.node_id !== d.node_id);
      });
    });
  }, []);

  return { devices, localInfo };
}
