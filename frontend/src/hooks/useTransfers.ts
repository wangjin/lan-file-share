import { useState, useEffect } from 'react';

export interface TransferTask {
  id: string;
  type: number; // 0=send, 1=receive
  state: number; // 0=pending, 1=transferring, 2=paused, 3=completed, 4=failed, 5=cancelled
  file_name: string;
  file_size: number;
  peer_id: string;
  peer_name: string;
  bytes_transferred: number;
  speed: number;
  created_at: string;
  completed_at?: string;
}

const stateNames = ['pending', 'transferring', 'paused', 'completed', 'failed', 'cancelled'] as const;
const typeNames = ['send', 'receive'] as const;

export function taskStateName(state: number): string {
  return stateNames[state] || 'unknown';
}

export function taskTypeName(type: number): string {
  return typeNames[type] || 'unknown';
}

export function useTransfers() {
  const [tasks, setTasks] = useState<TransferTask[]>([]);

  useEffect(() => {
    import('../../wailsjs/go/main/App').then(({ GetTasks }) => {
      GetTasks().then(setTasks);
    });

    import('../../wailsjs/runtime/runtime').then(({ EventsOn }) => {
      EventsOn('task:changed', (task: TransferTask) => {
        setTasks(prev => {
          const idx = prev.findIndex(t => t.id === task.id);
          if (idx >= 0) {
            const next = [...prev];
            next[idx] = task;
            return next;
          }
          return [...prev, task];
        });
      });
    });
  }, []);

  const sendFile = async (peerId: string) => {
    const { SelectAndSend } = await import('../../wailsjs/go/main/App');
    await SelectAndSend(peerId);
  };

  const cancelTask = async (taskId: string) => {
    const { CancelTask } = await import('../../wailsjs/go/main/App');
    await CancelTask(taskId);
  };

  return { tasks, sendFile, cancelTask };
}
