import React from 'react';
import { TransferTask, taskStateName } from '../hooks/useTransfers';
import { TransferItem } from './TransferItem';

interface Props {
  tasks: TransferTask[];
  peerId: string | null;
  onCancel: (id: string) => void;
}

export const TransferPanel: React.FC<Props> = ({ tasks, peerId, onCancel }) => {
  const filtered = peerId ? tasks.filter(t => t.peer_id === peerId) : tasks;
  const active = filtered.filter(t => taskStateName(t.state) === 'transferring');
  const waiting = filtered.filter(t => taskStateName(t.state) === 'pending');
  const done = filtered.filter(t => ['completed', 'failed', 'cancelled'].includes(taskStateName(t.state)));

  return (
    <div className="transfer-panel">
      {active.length > 0 && (
        <div className="section">
          <div className="section-title">传输中 ({active.length})</div>
          {active.map(t => <TransferItem key={t.id} task={t} onCancel={onCancel} />)}
        </div>
      )}
      {waiting.length > 0 && (
        <div className="section">
          <div className="section-title">等待中 ({waiting.length})</div>
          {waiting.map(t => <TransferItem key={t.id} task={t} onCancel={onCancel} />)}
        </div>
      )}
      {done.length > 0 && (
        <div className="section">
          <div className="section-title">已完成 ({done.length})</div>
          {done.map(t => <TransferItem key={t.id} task={t} onCancel={onCancel} />)}
        </div>
      )}
      {!active.length && !waiting.length && !done.length && (
        <div className="empty">暂无传输任务</div>
      )}
    </div>
  );
};
