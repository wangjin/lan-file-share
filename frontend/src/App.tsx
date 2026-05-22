import { useState, useCallback } from 'react';
import { useDevices } from './hooks/useDevices';
import { useTransfers } from './hooks/useTransfers';
import { useDragDrop } from './hooks/useDragDrop';
import { useUpdate } from './hooks/useUpdate';
import { SendPaths } from '../bindings/nearfy/app.js';
import {
  StartDownload,
  CancelDownload,
  InstallAndRestart,
  OpenReleasePage,
} from '../bindings/nearfy/internal/updater/service.js';
import { Sidebar } from './components/Sidebar';
import { TopBar } from './components/TopBar';
import { TransferPanel } from './components/TransferPanel';
import { UpdateToast } from './components/UpdateToast';
import { UpdateModal } from './components/UpdateModal';
import './App.css';

function App() {
  const { devices, localInfo } = useDevices();
  const { tasks, sendFile, respondReceive, cancelTask } = useTransfers();
  const { status, info, progress, error, ignoreUpdate } = useUpdate();
  const [selectedPeerId, setSelectedPeerId] = useState<string | null>(null);
  const [showModal, setShowModal] = useState(false);
  const selectedDevice = devices.find(d => d.node_id === selectedPeerId);

  const handleDrop = async (paths: string[]) => {
    if (!selectedPeerId) return;
    await SendPaths(selectedPeerId, paths);
  };

  const handleViewDetails = useCallback(() => {
    setShowModal(true);
  }, []);

  const handleStartDownload = useCallback(async () => {
    try {
      await StartDownload();
    } catch (e) {
      console.error('StartDownload failed:', e);
    }
  }, []);

  const handleCancel = useCallback(async () => {
    await CancelDownload();
    setShowModal(false);
  }, []);

  const handleInstall = useCallback(async () => {
    await InstallAndRestart();
  }, []);

  const handleManualDownload = useCallback(async () => {
    setShowModal(false);
    await OpenReleasePage();
  }, []);

  const handleDismissToast = useCallback(() => {
    ignoreUpdate();
  }, [ignoreUpdate]);

  const handleDismissModal = useCallback(() => {
    if (status !== 'downloading') {
      setShowModal(false);
    }
  }, [status]);

  const { isDragging, handlers: dragHandlers } = useDragDrop(handleDrop);

  return (
    <div className="app">
      <Sidebar
        devices={devices}
        localInfo={localInfo}
        selectedPeerId={selectedPeerId}
        onSelectDevice={setSelectedPeerId}
      />
      <div className="main">
        <TopBar
          device={selectedDevice}
          onSendFile={() => selectedPeerId && sendFile(selectedPeerId)}
        />
        <TransferPanel
          tasks={tasks}
          peerId={selectedPeerId}
          deviceName={selectedDevice?.name}
          onCancel={cancelTask}
          onRespond={respondReceive}
          isDragging={isDragging}
          dragHandlers={dragHandlers}
        />
      </div>
      {status === 'available' && !showModal && info && (
        <UpdateToast info={info} onViewDetails={handleViewDetails} onDismiss={handleDismissToast} />
      )}
      <UpdateModal
        visible={showModal}
        status={status}
        info={info}
        progress={progress}
        error={error}
        onStartDownload={handleStartDownload}
        onCancel={handleCancel}
        onInstall={handleInstall}
        onManualDownload={handleManualDownload}
        onClose={handleDismissModal}
      />
    </div>
  );
}

export default App;
