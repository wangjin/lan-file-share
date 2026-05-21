import React from 'react';
import { Device } from '../hooks/useDevices';

interface Props {
  device: Device;
  selected: boolean;
  onClick: () => void;
}

const osIcons: Record<string, string> = { darwin: '🍎', windows: '🪟', linux: '🐧' };

export const DeviceItem: React.FC<Props> = ({ device, selected, onClick }) => (
  <div className={`device-item ${selected ? 'selected' : ''}`} onClick={onClick}>
    <div className="device-avatar">{device.name[0]}</div>
    <div className="device-info">
      <div className="device-name">{device.name}</div>
      <div className="device-meta">{osIcons[device.os] || '💻'} {device.os} · {device.ip}</div>
    </div>
  </div>
);
