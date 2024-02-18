import ReactDOM from 'react-dom/client'
import App from './App.tsx'
import './index.css'
import SignalContextProvider from './rtc/SignalProvider.tsx'
import SubscriberContextProvider from './rtc/SubscriberProvider.tsx'
import MediaStreamProvider from './rtc/MediaStreamProvider.tsx'
import RoomContextProvider from './room/RoomProvider.tsx'
import RoomMediaStreamListProvider from './rtc/RoomMediaStreamListProvider.tsx'

ReactDOM.createRoot(document.getElementById('root')!).render(
  <SignalContextProvider>
    <MediaStreamProvider>
      <RoomMediaStreamListProvider>
        <SubscriberContextProvider>
          <RoomContextProvider>
            <App />
          </RoomContextProvider>
        </SubscriberContextProvider>
      </RoomMediaStreamListProvider>
    </MediaStreamProvider>
  </SignalContextProvider>
)

