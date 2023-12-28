import ReactDOM from 'react-dom/client'
import App from './App.tsx'
import './index.css'
import SignalContextProvider from './rtc/SignalProvider.tsx'
import SubscriberContextProvider from './rtc/SubscriberProvider.tsx'

ReactDOM.createRoot(document.getElementById('root')!).render(
  <SignalContextProvider>
    <SubscriberContextProvider>
      <App />
    </SubscriberContextProvider>
  </SignalContextProvider>
)

