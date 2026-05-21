import React, { useState } from 'react';
import { useDevices } from './hooks/useDevices';
import { useTransfers } from './hooks/useTransfers';
import { Sidebar } from './components/Sidebar';
import { TopBar } from './components/TopBar';
import { TransferPanel } from './components/TransferPanel';
import './App.css';

function App() {
  const { devices, localInfo } = useDevices();
  const { tasks, sendFile, respondReceive, cancelTask } = useTransfers();
  const [selectedPeerId, setSelectedPeerId] = useState<string | null>(null);
  const selectedDevice = devices.find(d => d.node_id === selectedPeerId);

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
          onCancel={cancelTask}
          onRespond={respondReceive}
        />
      </div>
    </div>
  );
}

export default App;
