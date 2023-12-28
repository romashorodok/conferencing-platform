import { PropsWithChildren, Dispatch, SetStateAction, createContext, useState, useEffect } from "react";
import { Signal } from "../App";

type SignalContextType = {
  signal: Signal,
  setSignal: Dispatch<SetStateAction<Signal>>
};

export const SignalContext = createContext<SignalContextType>(undefined as never)

function SignalContextProvider({ children }: PropsWithChildren<{}>) {
  const [signal, setSignal] = useState<Signal>(new Signal())

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
