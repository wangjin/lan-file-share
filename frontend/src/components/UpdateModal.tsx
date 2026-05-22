import React from 'react';
import Markdown from 'react-markdown';
import { UpdateInfo, DownloadProgress, UpdateStatus } from '../hooks/useUpdate';

interface Props {
  visible: boolean;
  status: UpdateStatus;
  info: UpdateInfo | null;
  progress: DownloadProgress | null;
  error: string | null;
  onStartDownload: () => void;
  onCancel: () => void;
  onInstall: () => void;
  onManualDownload: () => void;
  onClose: () => void;
}

function formatSpeed(bytesPerSec: number): string {
  if (bytesPerSec < 1024) return `${bytesPerSec.toFixed(0)} B/s`;
  if (bytesPerSec < 1024 * 1024) return `${(bytesPerSec / 1024).toFixed(1)} KB/s`;
  return `${(bytesPerSec / (1024 * 1024)).toFixed(1)} MB/s`;
}

function formatSize(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
}

export const UpdateModal: React.FC<Props> = ({
  visible, status, info, progress, error,
  onStartDownload, onCancel, onInstall, onManualDownload, onClose,
}) => {
  if (!visible || !info) return null;

  return (
    <div className="update-modal-overlay" onClick={status === 'downloading' ? undefined : onClose}>
      <div className="update-modal" onClick={e => e.stopPropagation()}>
        <div className="update-modal-header">
          <h3>更新到 {info.version}</h3>
          {status !== 'downloading' && (
            <button className="update-modal-close" onClick={onClose}>✕</button>
          )}
        </div>

        {info.body && (
          <div className="update-modal-body">
            <Markdown>{info.body}</Markdown>
          </div>
        )}

        {status === 'downloading' && progress && (
          <div className="update-modal-progress">
            <div className="progress-bar">
              <div className="progress-bar-fill" style={{ width: `${progress.percent}%` }} />
            </div>
            <div className="progress-info">
              <span>{progress.percent.toFixed(1)}%</span>
              <span>{formatSpeed(progress.speed)}</span>
              <span>{formatSize(progress.downloaded)} / {formatSize(progress.total)}</span>
            </div>
          </div>
        )}

        {status === 'error' && error && (
          <div className="update-modal-error">
            下载失败：{error}
          </div>
        )}

        <div className="update-modal-actions">
          {status === 'available' && (
            <>
              <button className="btn primary" onClick={onStartDownload}>立即更新</button>
              <button className="btn" onClick={onManualDownload}>手动下载</button>
              <button className="btn" onClick={onClose}>关闭</button>
            </>
          )}
          {status === 'downloading' && (
            <button className="btn" onClick={onCancel}>取消</button>
          )}
          {status === 'downloaded' && (
            <button className="btn primary" onClick={onInstall}>重启并安装</button>
          )}
          {status === 'error' && (
            <button className="btn primary" onClick={onManualDownload}>手动下载</button>
          )}
        </div>
      </div>
    </div>
  );
};
