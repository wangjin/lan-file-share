import React, { useEffect, useState } from 'react';
import { Device } from '../hooks/useDevices';
import { DeviceItem } from './DeviceItem';
import { GetVersion } from '../../bindings/nearfy/internal/updater/service.js';
import logo from '../assets/images/logo.png';

interface Props {
  devices: Device[];
  localInfo: { node_id: string; name: string; os: string } | null;
  selectedPeerId: string | null;
  onSelectDevice: (nodeId: string) => void;
}

export const Sidebar: React.FC<Props> = ({ devices, localInfo, selectedPeerId, onSelectDevice }) => {
  const [version, setVersion] = useState('');

  useEffect(() => {
    GetVersion().then((v: string) => setVersion(v));
  }, []);

  return (
    <div className="sidebar">
      <div className="local-info">
        <img className="avatar" src={logo} alt="" />
        <div className="info">
          <div className="name">{localInfo?.name}</div>
          <div className="status online">在线</div>
        </div>
      </div>
      <div className="device-section">
        <div className="section-title">在线设备 ({devices.length})</div>
        {devices.map(d => (
          <DeviceItem
            key={d.node_id}
            device={d}
            selected={d.node_id === selectedPeerId}
            onClick={() => onSelectDevice(d.node_id)}
          />
        ))}
      </div>
      {version && <div className="sidebar-version">v{version.replace(/^v/, '')}</div>}
    </div>
  );
};
