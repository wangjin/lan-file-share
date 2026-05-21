import React from 'react';
import { Device } from '../hooks/useDevices';

interface Props {
  device: Device | undefined;
  onSendFile: () => void;
}

export const TopBar: React.FC<Props> = ({ device, onSendFile }) => {
  if (!device) return <div className="topbar"><span className="hint">选择一个设备开始传输</span></div>;
  return (
    <div className="topbar">
      <div className="peer-info">
        <span className="peer-name">{device.name}</span>
        <span className="peer-ip">{device.ip}</span>
      </div>
      <div className="actions">
        <button className="btn-primary" onClick={onSendFile}>发送文件</button>
      </div>
    </div>
  );
};
