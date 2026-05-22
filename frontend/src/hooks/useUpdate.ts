import { useState, useEffect, useCallback } from 'react';
import { Events } from '@wailsio/runtime';

export interface UpdateInfo {
  version: string;
  body: string;
  downloadUrl: string;
  assetSize: number;
}

export interface DownloadProgress {
  downloaded: number;
  total: number;
  percent: number;
  speed: number;
}

export type UpdateStatus = 'idle' | 'available' | 'downloading' | 'downloaded' | 'error';

interface UpdateState {
  status: UpdateStatus;
  info: UpdateInfo | null;
  progress: DownloadProgress | null;
  error: string | null;
}

export function useUpdate() {
  const [state, setState] = useState<UpdateState>({
    status: 'idle',
    info: null,
    progress: null,
    error: null,
  });
  const [ignoredVersion, setIgnoredVersion] = useState<string | null>(() => {
    return localStorage.getItem('ignoredUpdateVersion');
  });

  useEffect(() => {
    const off1 = Events.On('update:available', (ev: any) => {
      const data = ev.data as UpdateInfo;
      if (data.version === ignoredVersion) return;
      setState({ status: 'available', info: data, progress: null, error: null });
    });

    const off2 = Events.On('update:progress', (ev: any) => {
      setState(prev => ({
        ...prev,
        status: 'downloading',
        progress: ev.data as DownloadProgress,
      }));
    });

    const off3 = Events.On('update:downloaded', () => {
      setState(prev => ({ ...prev, status: 'downloaded', progress: null }));
    });

    const off4 = Events.On('update:error', (ev: any) => {
      setState(prev => ({
        ...prev,
        status: 'error',
        error: (ev.data as { error: string }).error,
      }));
    });

    return () => {
      off1();
      off2();
      off3();
      off4();
    };
  }, [ignoredVersion]);

  const ignoreUpdate = useCallback(() => {
    if (state.info) {
      const v = state.info.version;
      setIgnoredVersion(v);
      localStorage.setItem('ignoredUpdateVersion', v);
      setState({ status: 'idle', info: null, progress: null, error: null });
    }
  }, [state.info]);

  return {
    ...state,
    ignoredVersion,
    ignoreUpdate,
  };
}
