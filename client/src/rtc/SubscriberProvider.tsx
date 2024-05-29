import { PropsWithChildren, createContext, useState, useEffect, Dispatch, SetStateAction, useContext } from "react";
import { RTCEngine } from "../App";
import { RoomMediaStreamListContext } from "./RoomMediaStreamListProvider";

type SubscriberContextType = {
  peerContext: RTCEngine | null,
  setPeerContext: Dispatch<SetStateAction<RTCEngine | null>>
}

export const SubscriberContext = createContext<SubscriberContextType>(undefined!)

function SubscriberContextProvider({ children }: PropsWithChildren<{}>) {
  const [peerContext, setPeerContext] = useState<RTCEngine | null>(null)
  const { setRoomMediaStream } = useContext(RoomMediaStreamListContext)

  useEffect(() => {
    return () => {
      if (peerContext) {
        setRoomMediaStream({ action: 'clear', payload: undefined! })
        peerContext.close()
      }
    }
  }, [peerContext])

  return (
    <SubscriberContext.Provider value={{ peerContext, setPeerContext }}>
      {children}
    </SubscriberContext.Provider>
  )
}

export default SubscriberContextProvider;
