import React from 'react';
import { TransferTask, taskStateName, taskTypeName } from '../hooks/useTransfers';

interface Props {
  task: TransferTask;
  onCancel: (id: string) => void;
}

function formatSize(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  return `${(bytes / 1024 / 1024).toFixed(1)} MB`;
}

export const TransferItem: React.FC<Props> = ({ task, onCancel }) => {
  const state = taskStateName(task.state);
  const progress = task.file_size > 0 ? (task.bytes_transferred / task.file_size * 100).toFixed(1) : '0';

  return (
    <div className={`transfer-item state-${state}`}>
      <div className="transfer-header">
        <span className="direction">{taskTypeName(task.type) === 'send' ? '↑' : '↓'}</span>
        <span className="filename">{task.file_name}</span>
        {(state === 'transferring' || state === 'pending') && (
          <button className="btn-cancel" onClick={() => onCancel(task.id)}>✕</button>
        )}
      </div>
      <div className="transfer-meta">
        {state === 'transferring' && `${formatSize(task.bytes_transferred)} / ${formatSize(task.file_size)} · ${formatSize(task.speed)}/s`}
        {state === 'pending' && `${formatSize(task.file_size)} · 等待传输...`}
        {state === 'completed' && `${formatSize(task.file_size)} · 完成`}
        {state === 'failed' && `${formatSize(task.file_size)} · 失败`}
        {state === 'cancelled' && `${formatSize(task.file_size)} · 已取消`}
      </div>
      {state === 'transferring' && (
        <div className="progress-bar">
          <div className="progress-fill" style={{ width: `${progress}%` }} />
        </div>
      )}
    </div>
  );
};
