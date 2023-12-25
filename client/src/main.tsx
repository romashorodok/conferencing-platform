import ReactDOM from 'react-dom/client'
import App from './App.tsx'
import './index.css'
import PeerConnectionProvider from './PeerConnectionProvider.tsx'

ReactDOM.createRoot(document.getElementById('root')!).render(
  <PeerConnectionProvider>
    <App />
  </PeerConnectionProvider>
)
