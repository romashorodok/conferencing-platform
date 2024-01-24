import { PropsWithChildren, createContext, useEffect, useState } from "react";

type MediaStreamContextType = {
  mediaStream: Promise<MediaStream>
  mediaStreamReady: boolean
};

export const MediaStreamContext = createContext<MediaStreamContextType>(undefined!)

// const videoStream = navigator.mediaDevices.getUserMedia({
//   video: true,
// })
//
// const audioStream = navigator.mediaDevices.getUserMedia({
//   audio: true,
//   // NOTE: Firefox not support low resolutions
//   // video: {
//   //   width: { exact: 448 },
//   //   height: { exact: 216 }
//   // },
//   // audio: true
// });

const mediaStream = navigator.mediaDevices.getUserMedia({
  audio: true,
  video: true,
})

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
