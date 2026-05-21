import { useState, useEffect } from 'react';
import { Events } from '@wailsio/runtime';
import { GetTasks, SelectAndSend, RespondReceive, ChooseSavePath, CancelTask } from '../../bindings/lan-file-share/app.js';
import { TransferTask } from '../../bindings/lan-file-share/internal/model/models.js';

export type { TransferTask };

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
    GetTasks().then((t: (TransferTask | null)[]) => {
      setTasks(t.filter((x): x is TransferTask => x !== null));
    });

    Events.On('task:changed', (ev: any) => {
      const task = ev.data as TransferTask;
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
  }, []);

  const sendFile = async (peerId: string) => {
    await SelectAndSend(peerId);
  };

  const respondReceive = async (taskId: string, accept: boolean) => {
    let savePath = '';
    if (accept) {
      const task = tasks.find(t => t.id === taskId);
      savePath = await ChooseSavePath(task?.file_name || 'file');
      if (!savePath) return;
    }
    await RespondReceive(taskId, accept, savePath);
  };

  const cancelTask = async (taskId: string) => {
    await CancelTask(taskId);
  };

  return { tasks, sendFile, respondReceive, cancelTask };
}
