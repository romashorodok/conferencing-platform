import ReactDOM from 'react-dom/client'
import App from './App.tsx'
import './index.css'
import SignalContextProvider from './rtc/SignalProvider.tsx'
import SubscriberContextProvider from './rtc/SubscriberProvider.tsx'
import MediaStreamProvider from './rtc/MediaStreamProvider.tsx'
import RoomNotifierContextProvider from './room/RoomProvider.tsx'
import RoomMediaStreamListProvider from './rtc/RoomMediaStreamListProvider.tsx'
import { Outlet, RouterProvider, createBrowserRouter } from 'react-router-dom'
import AppLayout, { DialogWindow, SignInForm } from './AppLayout.tsx'
import RoomPage from './routes/r/RoomPage.tsx'
import ControlsProvider from './rtc/ControlsProvider.tsx'
import { AuthContextProvider, useAuth } from './rtc/AuthProvider.tsx'
import { PropsWithChildren, useState } from 'react'

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

function AuthWall({ children }: PropsWithChildren) {
  const { authenticated } = useAuth()
  const [open, setOpen] = useState(false)

  if (!authenticated) {
    const button = <button
      className="Button box-border w-full inline-flex h-[35px] items-center justify-center rounded-[4px] px-[15px] font-medium leading-none cursor-pointer focus:outline-none mt-[10px]" onClick={() => setOpen(true)}>
      SignUp
    </button>

    return <div className="flex min-h-screen min-w-screen flex-col m-auto justify-center place-items-center">
      <div className="w-[200px]">
        <h1>Sign Up to continue!</h1>

        <DialogWindow open={open} setOpen={setOpen} button={button} title={<p>Sign Up</p>} description={<></>} content={<SignInForm setOpen={setOpen} />} />
      </div>
    </div>
  }

  return children
}

const router = createBrowserRouter([
  {
    path: '/',
    element:
      <AuthContextProvider>
        <AuthWall>
          <Layout />
        </AuthWall>
      </AuthContextProvider>
    ,
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
