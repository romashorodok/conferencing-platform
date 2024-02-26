import { PropsWithChildren, createContext, useState, useEffect, Dispatch, SetStateAction } from "react";
import { Signal, SignalEvent } from "../App";

type SignalContextType = {
  signal: Signal | null,
  setSignal: Dispatch<SetStateAction<Signal | null>>
};

export const SignalContext = createContext<SignalContextType>(undefined as never)

// const SIGNAL_ROOM = import.meta.env.VITE_DEFAULT_SIGNAL_ROOM;

function SignalContextProvider({ children }: PropsWithChildren<{}>) {
  const [signal, setSignal] = useState<Signal | null>(null)

  useEffect(() => {
    return () => {
      if (signal) {
        signal.removeAllListeners(SignalEvent.Offer)
        signal.destroy()
      }
    }
  }, [signal])
  // async function setSignalServer(roomId: string) {
  //   setSignal(new Signal(`${MEDIA_SERVER_WS}/rooms/${roomId}`))
  //
  // }

  // useEffect(() => {
  //   if (signal) {
  //     signal.connect()
  //     console.log(`Signaling server: ${signal.uri}`)
  //   }
  //   return () => {
  //     if (signal) {
  //       signal.destroy()
  //     }
  //   }
  // }, [signal])

  return (
    <SignalContext.Provider value={{ signal, setSignal }}>
      {children}
    </SignalContext.Provider>
  )
}

export default SignalContextProvider;
