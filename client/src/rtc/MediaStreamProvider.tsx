import { PropsWithChildren, createContext, useContext, useEffect, useState } from "react";
import { isChromiumBased } from "../helpers";

type MediaStreamContextType = {
  mediaStream: MediaStream
  mediaStreamReady: boolean

  setAudioMute: (mute: boolean) => Promise<void>
  setVideoMute: (mute: boolean) => Promise<void>

  startFaceDetection: () => void
  startNormal: () => void
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

async function setVideoMute(mute: boolean) {
  const stream = await videoStream
  stream.getVideoTracks().forEach(t => t.enabled = !mute)
}

async function setAudioMute(mute: boolean) {
  const stream = await audioStream
  stream.getAudioTracks().forEach(t => t.enabled = !mute)
}

const defaultMediaStream =
  new Promise<MediaStream>(async (resolve, reject) => {
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

const faceDetectionMediaStream = (faceDetectorWorker: Worker): Promise<MediaStream> =>
  new Promise(async (resolve, reject) => {
    if (!isChromiumBased()) {
      reject("support only chromium based browsers")
    }

    try {
      let stream = await defaultMediaStream
      stream = stream.clone()

      const [track] = stream.getVideoTracks()
      stream.removeTrack(track)

      const { readable } = new MediaStreamTrackProcessor({ track })
      const localTrack = new MediaStreamTrackGenerator({ kind: 'video' })
      const { writable } = localTrack

      faceDetectorWorker.postMessage({
        readable,
        writable,
      }, [readable, writable])

      stream.addTrack(localTrack)

      resolve(stream)
    } catch (e) {
      reject(e)
    }
  })

async function startFaceDetectionStream(
): Promise<{
  stream: MediaStream,
  worker: Worker
}> {
  const faceDetectorWorker = new Worker(new URL('workers/faceDetector.js', import.meta.url), { type: 'classic' })
  return {
    stream: await faceDetectionMediaStream(faceDetectorWorker),
    worker: faceDetectorWorker
  }
}

function MediaStreamProvider({ children }: PropsWithChildren<{}>) {
  const [mediaStream, setMediaStream] = useState<MediaStream>()
  const [mediaStreamReady, setMediaStreamReady] = useState(false)
  const [worker, setWorker] = useState<Worker>()

  function startNormal() {
    defaultMediaStream
      .then(stream => {
        setMediaStream(stream)
        setMediaStreamReady(true)
      })
  }

  function startFaceDetection() {
    if (isChromiumBased()) {
      startFaceDetectionStream()
        .then(({ stream, worker }) => {
          setMediaStream(stream)
          setMediaStreamReady(true)
          setWorker(worker)
        })
    } else {
      console.warn("Face detection work only in chromium based browsers")
      startNormal()
    }
  }

  useEffect(() => {
    startNormal()
  }, [])

  useEffect(() => {
    console.log("Cammer changed")
    return () => {
      if (worker) {
        worker.terminate()
        setWorker(undefined)
      }
    }
  }, [mediaStream])

  return (
    <MediaStreamContext.Provider value={{ mediaStream, mediaStreamReady, setAudioMute, setVideoMute, startNormal, startFaceDetection }}>
      {children}
    </MediaStreamContext.Provider>
  )
}

export default MediaStreamProvider;
