import React from 'react';
import { TransferTask, taskStateName } from '../hooks/useTransfers';
import { TransferItem } from './TransferItem';

interface Props {
  tasks: TransferTask[];
  peerId: string | null;
  deviceName: string | undefined;
  onCancel: (id: string) => void;
  onRespond: (id: string, accept: boolean) => void;
  isDragging: boolean;
  dragHandlers: {
    onDragEnter: (e: React.DragEvent) => void;
    onDragOver: (e: React.DragEvent) => void;
    onDragLeave: (e: React.DragEvent) => void;
  };
}

export const TransferPanel: React.FC<Props> = ({
  tasks, peerId, deviceName, onCancel, onRespond, isDragging, dragHandlers,
}) => {
  const filtered = peerId ? tasks.filter(t => t.peer_id === peerId) : tasks;
  const active = filtered.filter(t => taskStateName(t.state) === 'transferring');
  const waiting = filtered.filter(t => taskStateName(t.state) === 'pending');
  const done = filtered.filter(t => ['completed', 'failed', 'cancelled'].includes(taskStateName(t.state)));

  return (
    <div
      className="transfer-panel"
      data-file-drop-target
      {...dragHandlers}
    >
      {active.length > 0 && (
        <div className="section">
          <div className="section-title">传输中 ({active.length})</div>
          {active.map(t => <TransferItem key={t.id} task={t} onCancel={onCancel} onRespond={onRespond} />)}
        </div>
      )}
      {waiting.length > 0 && (
        <div className="section">
          <div className="section-title">等待中 ({waiting.length})</div>
          {waiting.map(t => <TransferItem key={t.id} task={t} onCancel={onCancel} onRespond={onRespond} />)}
        </div>
      )}
      {done.length > 0 && (
        <div className="section">
          <div className="section-title">已完成 ({done.length})</div>
          {done.map(t => <TransferItem key={t.id} task={t} onCancel={onCancel} onRespond={onRespond} />)}
        </div>
      )}
      {!active.length && !waiting.length && !done.length && !isDragging && (
        <div className="empty">暂无传输任务</div>
      )}
      {isDragging && (
        <div className={`dropzone-overlay ${peerId ? 'active' : 'disabled'}`}>
          {peerId ? (
            <span>释放以发送到 {deviceName}</span>
          ) : (
            <span>请先选择一个设备</span>
          )}
        </div>
      )}
    </div>
  );
};
