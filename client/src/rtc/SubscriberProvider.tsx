import { PropsWithChildren, Dispatch, SetStateAction, createContext, useState, useEffect } from "react";
import { RTCEngine } from "../App";

type SubscriberContextType = {
  subscriber: RTCEngine,
  setSubscriber: Dispatch<SetStateAction<RTCEngine>>
};

export const SubscriberContext = createContext<SubscriberContextType>(undefined as never)

function SubscriberContextProvider({ children }: PropsWithChildren<{}>) {
  const [subscriber, setSubscriber] = useState<RTCEngine>(new RTCEngine({
    iceServers: [
      {
        urls: 'stun:stun.l.google.com:19302'
      },
    ]
  }))

  useEffect(() => {
    return () => {
      if (subscriber) {
        subscriber.close()
      }
    }
  }, [subscriber])

  return (
    <SubscriberContext.Provider value={{ subscriber, setSubscriber }}>
      {children}
    </SubscriberContext.Provider>
  )
}

export default SubscriberContextProvider;
