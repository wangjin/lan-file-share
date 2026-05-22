import React from 'react';
import { UpdateInfo } from '../hooks/useUpdate';

interface Props {
  info: UpdateInfo;
  onViewDetails: () => void;
  onDismiss: () => void;
}

export const UpdateToast: React.FC<Props> = ({ info, onViewDetails, onDismiss }) => (
  <div className="update-toast">
    <div className="update-toast-content">
      <div className="update-toast-title">发现新版本 {info.version}</div>
      <div className="update-toast-actions">
        <button className="update-toast-btn primary" onClick={onViewDetails}>查看详情</button>
        <button className="update-toast-btn" onClick={onDismiss}>忽略</button>
      </div>
    </div>
  </div>
);
