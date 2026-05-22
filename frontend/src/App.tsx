import { useState } from 'react';
import { useDevices } from './hooks/useDevices';
import { useTransfers } from './hooks/useTransfers';
import { useDragDrop } from './hooks/useDragDrop';
import { SendPaths } from '../bindings/lan-file-share/app.js';
import { Sidebar } from './components/Sidebar';
import { TopBar } from './components/TopBar';
import { TransferPanel } from './components/TransferPanel';
import './App.css';

function App() {
  const { devices, localInfo } = useDevices();
  const { tasks, sendFile, respondReceive, cancelTask } = useTransfers();
  const [selectedPeerId, setSelectedPeerId] = useState<string | null>(null);
  const selectedDevice = devices.find(d => d.node_id === selectedPeerId);

  const handleDrop = async (paths: string[]) => {
    if (!selectedPeerId) return;
    await SendPaths(selectedPeerId, paths);
  };

  const { isDragging, handlers: dropHandlers } = useDragDrop(handleDrop, !!selectedPeerId);

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
          dropHandlers={dropHandlers}
        />
      </div>
    </div>
  );
}

export default App;
