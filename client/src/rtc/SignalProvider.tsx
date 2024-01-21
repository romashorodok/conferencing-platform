import { PropsWithChildren, Dispatch, SetStateAction, createContext, useState, useEffect } from "react";
import { Signal } from "../App";

type SignalContextType = {
  signal: Signal,
  setSignal: Dispatch<SetStateAction<Signal>>
};

export const SignalContext = createContext<SignalContextType>(undefined as never)

const SIGNAL_ROOM = import.meta.env.VITE_DEFAULT_SIGNAL_ROOM;

function SignalContextProvider({ children }: PropsWithChildren<{}>) {
  console.log(SIGNAL_ROOM)
  const [signal, setSignal] = useState<Signal>(new Signal(SIGNAL_ROOM))

  useEffect(() => {
    if (signal) {
      signal.connect()
    }
  }, [signal])

  return (
    <SignalContext.Provider value={{ signal, setSignal }}>
      {children}
    </SignalContext.Provider>
  )
}

export default SignalContextProvider;
