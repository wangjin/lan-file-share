import { useState, useEffect, useCallback } from 'react';
import { Events } from '@wailsio/runtime';

export function useDragDrop(onDrop: (paths: string[]) => void) {
  const [isDragging, setIsDragging] = useState(false);

  useEffect(() => {
    const off = Events.On('files-dropped', (ev: any) => {
      setIsDragging(false);
      const files: string[] = ev.data?.files || [];
      if (files.length > 0) {
        onDrop(files);
      }
    });
    return () => { off(); };
  }, [onDrop]);

  const handleDragEnter = useCallback((e: React.DragEvent) => {
    e.preventDefault();
    if (e.dataTransfer.types.includes('Files')) {
      setIsDragging(true);
    }
  }, []);

  const handleDragOver = useCallback((e: React.DragEvent) => {
    e.preventDefault();
  }, []);

  const handleDragLeave = useCallback((e: React.DragEvent) => {
    e.preventDefault();
    const related = e.relatedTarget as Node | null;
    if (related && e.currentTarget.contains(related)) return;
    setIsDragging(false);
  }, []);

  return {
    isDragging,
    handlers: {
      onDragEnter: handleDragEnter,
      onDragOver: handleDragOver,
      onDragLeave: handleDragLeave,
    },
  };
}
