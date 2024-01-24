import { PropsWithChildren, createContext, useEffect, useState } from "react";
import { isChromiumBased } from "../helpers";

type MediaStreamContextType = {
  mediaStream: Promise<MediaStream>
  mediaStreamReady: boolean

  setAudioMute: (mute: boolean) => Promise<void>
  setVideoMute: (mute: boolean) => Promise<void>
};

export const MediaStreamContext = createContext<MediaStreamContextType>(undefined!)

const videoStream = navigator.mediaDevices.getUserMedia({
  video: true,
  // NOTE: Firefox not support low resolutions
  // video: {
  //   width: { exact: 448 },
  //   height: { exact: 216 }
  // },
})

const audioConfig: MediaTrackConstraints = {
  noiseSuppression: true,
  echoCancellation: false,
  autoGainControl: false,
  sampleRate: 48000,
}

if (isChromiumBased()) {
  // @ts-ignore
  audioConfig['googAutoGainControl'] = false
}

const audioStream = navigator.mediaDevices.getUserMedia({
  audio: audioConfig,
});

const mediaStream = new Promise<MediaStream>(async (resolve, reject) => {
  const stream = new MediaStream()
  const tracks: Array<MediaStreamTrack> = []

  try {
    const video = await videoStream
    const [track] = video.getVideoTracks()
    tracks.push(track)
  } catch (err) {
    console.error(err)
  }

  try {
    const audio = await audioStream
    const [track] = audio.getAudioTracks()
    tracks.push(track)
  } catch (err) {
    console.error(err)
  }

  if (tracks.length <= 0)
    reject(new Error("Empty tracks"))

  tracks.forEach(t => stream.addTrack(t))
  resolve(stream)
})

async function setVideoMute(mute: boolean) {
  const stream = await videoStream
  stream.getVideoTracks().forEach(t => t.enabled = !mute)
}

async function setAudioMute(mute: boolean) {
  const stream = await audioStream
  stream.getAudioTracks().forEach(t => t.enabled = !mute)
}

function MediaStreamProvider({ children }: PropsWithChildren<{}>) {
  const [mediaStreamReady, setMediaStreamReady] = useState(false)

  useEffect(() => {
    mediaStream.then(_ => setMediaStreamReady(true))
  }, [])

  return (
    <MediaStreamContext.Provider value={{ mediaStream, mediaStreamReady, setAudioMute, setVideoMute }}>
      {children}
    </MediaStreamContext.Provider>
  )
}

export default MediaStreamProvider;
