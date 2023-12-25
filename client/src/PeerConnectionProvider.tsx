import { PropsWithChildren, Dispatch, SetStateAction, createContext, useState, useEffect } from "react";

type PeerConnectionContextType = {
  peerConnection: RTCPeerConnection,
  setPeerConnection: Dispatch<SetStateAction<RTCPeerConnection>>
};

export const PeerConnectionContext = createContext<PeerConnectionContextType>(undefined as never)

function PeerConnectionProvider({ children }: PropsWithChildren<{}>) {
  const [peerConnection, setPeerConnection] = useState<RTCPeerConnection>(undefined as never)

  useEffect(() => {
    if (!peerConnection) {
      setPeerConnection(new RTCPeerConnection())
    }

    return () => {
      if (peerConnection) {
        peerConnection.close()
      }
    }
  }, [peerConnection])

  return (
    <PeerConnectionContext.Provider value={{ peerConnection, setPeerConnection }}>
      {children}
    </PeerConnectionContext.Provider>
  )
}

export default PeerConnectionProvider;
