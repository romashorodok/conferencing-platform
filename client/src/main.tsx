import ReactDOM from 'react-dom/client'
import App from './App.tsx'
import './index.css'
import SignalContextProvider from './rtc/SignalProvider.tsx'
import SubscriberContextProvider from './rtc/SubscriberProvider.tsx'
import MediaStreamProvider from './rtc/MediaStreamProvider.tsx'

ReactDOM.createRoot(document.getElementById('root')!).render(
  <SignalContextProvider>
    <MediaStreamProvider>
      <SubscriberContextProvider>
        <App />
      </SubscriberContextProvider>
    </MediaStreamProvider>
  </SignalContextProvider>
)

