import ReactDOM from 'react-dom/client'
import App from './App.tsx'
import './index.css'
import SignalContextProvider from './rtc/SignalProvider.tsx'
import SubscriberContextProvider from './rtc/SubscriberProvider.tsx'
import MediaStreamProvider from './rtc/MediaStreamProvider.tsx'
import RoomNotifierContextProvider from './room/RoomProvider.tsx'
import RoomMediaStreamListProvider from './rtc/RoomMediaStreamListProvider.tsx'
import { Outlet, RouterProvider, createBrowserRouter } from 'react-router-dom'
import AppLayout from './AppLayout.tsx'
import RoomPage from './routes/r/RoomPage.tsx'
import ControlsProvider from './rtc/ControlsProvider.tsx'

function Layout() {
  return (
    <SignalContextProvider>
      <MediaStreamProvider>
        <RoomMediaStreamListProvider>
          <SubscriberContextProvider>
            <RoomNotifierContextProvider>
              <ControlsProvider>
                <AppLayout>
                  <Outlet />
                </AppLayout>
              </ControlsProvider>
            </RoomNotifierContextProvider>
          </SubscriberContextProvider>
        </RoomMediaStreamListProvider>
      </MediaStreamProvider>
    </SignalContextProvider>
  )
}

const router = createBrowserRouter([
  {
    path: '/',
    element: <Layout />,
    children: [
      {
        path: '',
        element: <App />
      },
      {
        path: 'r/:roomID',
        element: <RoomPage />,
      }
    ]
  },
])

ReactDOM.createRoot(document.getElementById('root')!).render(
  <RouterProvider router={router} />
)
