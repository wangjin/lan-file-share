import React from 'react';
import { TransferTask, taskStateName, taskTypeName } from '../hooks/useTransfers';

interface Props {
  task: TransferTask;
  onCancel: (id: string) => void;
  onRespond: (id: string, accept: boolean) => void;
}

function formatSize(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  return `${(bytes / 1024 / 1024).toFixed(1)} MB`;
}

export const TransferItem: React.FC<Props> = ({ task, onCancel, onRespond }) => {
  const state = taskStateName(task.state);
  const isReceive = taskTypeName(task.type) === 'receive';
  const isPendingReceive = state === 'pending' && isReceive;
  const progress = task.file_size > 0 ? (task.bytes_transferred / task.file_size * 100).toFixed(1) : '0';

  return (
    <div className={`transfer-item state-${state}`}>
      <div className="transfer-header">
        <span className="direction">{isReceive ? '↓' : '↑'}</span>
        <span className="filename">{task.file_name}</span>
        {isPendingReceive ? (
          <div className="receive-actions">
            <button className="btn-accept" onClick={() => onRespond(task.id, true)}>接收</button>
            <button className="btn-reject" onClick={() => onRespond(task.id, false)}>拒绝</button>
          </div>
        ) : (
          (state === 'transferring' || (state === 'pending' && !isReceive)) && (
            <button className="btn-cancel" onClick={() => onCancel(task.id)}>✕</button>
          )
        )}
      </div>
      <div className="transfer-meta">
        {isPendingReceive && `${task.peer_name || '未知设备'} · ${formatSize(task.file_size)} · 等待确认...`}
        {!isPendingReceive && state === 'transferring' && `${formatSize(task.bytes_transferred)} / ${formatSize(task.file_size)} · ${formatSize(task.speed)}/s`}
        {!isPendingReceive && state === 'pending' && `${formatSize(task.file_size)} · 等待传输...`}
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
