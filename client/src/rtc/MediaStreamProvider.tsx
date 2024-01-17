import { PropsWithChildren, createContext, useEffect, useState } from "react";

type MediaStreamContextType = {
  mediaStream: Promise<MediaStream>
  mediaStreamReady: boolean
};

export const MediaStreamContext = createContext<MediaStreamContextType>(undefined!)

const mediaStream = navigator.mediaDevices.getUserMedia({ video: true, audio: true })

function MediaStreamProvider({ children }: PropsWithChildren<{}>) {
  const [mediaStreamReady, setMediaStreamReady] = useState(false)

  useEffect(() => {
    mediaStream.then(_ => setMediaStreamReady(true))
  }, [])

  return (
    <MediaStreamContext.Provider value={{ mediaStream, mediaStreamReady }}>
      {children}
    </MediaStreamContext.Provider>
  )
}

export default MediaStreamProvider;
