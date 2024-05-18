import { PropsWithChildren, createContext, useEffect, useState } from "react";
import { isChromiumBased, isFireFox } from "../helpers";
import { Mutex } from "../App";

type MediaStreamContextType = {
  mediaStream: MediaStream
  mediaStreamReady: boolean
  onPageMountMediaStreamMutex: Mutex | null

  startFaceDetection: (unlock?: Promise<() => void>) => Promise<void>
  startNormal: (unlock?: Promise<() => void>) => Promise<void>
};

export const MediaStreamContext = createContext<MediaStreamContextType>(undefined!)

// Firefox only
async function videoScaleCanvas(streamPromise: Promise<MediaStream>, width: number, height: number): Promise<MediaStream> {
  const video = document.createElement('video');
  video.width = width
  video.height = height
  const canvas = document.createElement('canvas');

  // @ts-ignore
  const offscreen: OffscreenCanvas = canvas.transferControlToOffscreen()
  offscreen.width = width
  offscreen.height = height
  const ctx = offscreen.getContext('2d');

  const stream = await streamPromise
  const caps = stream.getVideoTracks()[0].getSettings()

  console.info(`videoScaleCanvas enabled | inbound: ${caps.width}x${caps.height}, outbound: ${width}x${height}`)

  const videoTrack = new Array<MediaStreamTrack>(stream.getVideoTracks()[0])
  const inboundStream = new MediaStream(videoTrack)
  video.srcObject = inboundStream

  function drawAndDownscale() {
    ctx?.clearRect(0, 0, canvas.width, canvas.height);
    ctx?.drawImage(video, 0, 0, canvas.width, canvas.height);
    requestAnimationFrame(drawAndDownscale);
  }

  video.addEventListener('play', drawAndDownscale);
  video.play()

  return canvas.captureStream().clone()
}

let videoStream: Promise<MediaStream>
if (isChromiumBased()) {
  // const downScaleWorker = new Worker(new URL('workers/downScale.js', import.meta.url), { type: 'classic' })

  // NOTE: Selecting video may be work only on localhost
  const inbound = navigator.mediaDevices.getUserMedia({
    video: {
      width: 320,
      height: 180,
    },
    // video: {
    //   width: { exact: 320 },
    //   height: { exact: 180 }
    // },
  })

  videoStream = inbound

  // videoStream = inbound.then(async defaultStream => {
  //   const stream = defaultStream.clone()
  //   stream.getTracks().forEach(t => {
  //     t.enabled = true
  //     const track = t.clone()
  //     stream.removeTrack(t)
  //     stream.addTrack(track)
  //   })
  //
  //   const [track] = stream.getVideoTracks()
  //   // NOTE: need remove origin track it will be replaced by sink track
  //   stream.removeTrack(track)
  //
  //   const { readable } = new MediaStreamTrackProcessor({ track })
  //   const localTrack = new MediaStreamTrackGenerator({ kind: 'video' })
  //   const { writable } = localTrack
  //
  //   downScaleWorker.postMessage({
  //     width: 320,
  //     height: 180,
  //     readable,
  //     writable,
  //   }, [readable, writable])
  //
  //   stream.addTrack(localTrack)
  //
  //   stream.getTracks().forEach(t => t.enabled = false)
  //
  //   return stream
  // })

  // videoStream = videoScaleCanvas(inbound, 320, 180)
} else if (isFireFox()) {
  // NOTE: Firefox not support low resolutions
  // This lead that I need high performance, when I do the filter of the stream
  const inbound = navigator.mediaDevices.getUserMedia({ video: true })
  // NOTE: workaround of that problem. This not lead to high cpu usage
  // But also the drawing may be in a web-worker
  videoStream = videoScaleCanvas(inbound, 320, 180)

} else {
  throw new Error("unsupported browser target")
}

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

export async function setVideoMute(stream: MediaStream, mute: boolean) {
  stream.getVideoTracks().forEach(t => t.enabled = !mute)
}

export async function setAudioMute(stream: MediaStream, mute: boolean) {
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

// NOTE: Insertable stream. Chrome only feature
const faceDetectionMediaStream = (faceDetectorWorker: Worker): Promise<MediaStream> =>
  new Promise(async (resolve, reject) => {
    if (!isChromiumBased()) {
      reject("support only chromium based browsers")
    }

    try {
      let defaultStream = await defaultMediaStream.then(m => {
        console.log(m)
        return m
      })
      const stream = defaultStream.clone()
      stream.getTracks().forEach(t => {
        t.enabled = true
        const track = t.clone()
        stream.removeTrack(t)
        stream.addTrack(track)
      })

      const [track] = stream.getVideoTracks()
      // NOTE: need remove origin track it will be replaced by sink track
      stream.removeTrack(track)

      const { readable } = new MediaStreamTrackProcessor({ track })
      const localTrack = new MediaStreamTrackGenerator({ kind: 'video' })
      const { writable } = localTrack

      faceDetectorWorker.postMessage({
        readable,
        writable,
      }, [readable, writable])

      stream.addTrack(localTrack)

      stream.getTracks().forEach(t => t.enabled = false)

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
  const [onPageMountMediaStreamMutex, setOnPageMountMediaStreamMutex] = useState<Mutex | null>(null)

  async function startNormal(unlock?: Promise<() => void>) {
    // const unlock = await mediaStreamMutex.lock()
    defaultMediaStream
      .then(async stream => {
        setVideoMute(stream, true)
        setAudioMute(stream, true)

        setMediaStream(stream);
        setMediaStreamReady(true);
        if (unlock)
          (await unlock)()
      })
  }

  async function startFaceDetection(unlock?: Promise<() => void>) {
    if (isChromiumBased()) {
      startFaceDetectionStream()
        .then(async ({ stream, worker }) => {
          setMediaStream(stream);
          setMediaStreamReady(true);
          setWorker(worker);
          if (unlock)
            (await unlock)()
        })
    } else {
      console.warn("Face detection work only in chromium based browsers")
      startNormal()
    }
  }

  useEffect(() => {
    const mu = new Mutex()
    const unlock = mu.lock()

    if (!navigator.mediaDevices?.enumerateDevices) {
      console.log("enumerateDevices() not supported.");
    } else {
      // List cameras and microphones.
      navigator.mediaDevices
        .enumerateDevices()
        .then((devices) => {
          devices.forEach((device) => {
            console.log(`${device.kind}: ${device.label} id = ${device.deviceId}`);
          });
        })
        .catch((err) => {
          console.error(`${err.name}: ${err.message}`);
        });
    }

    startNormal(unlock)
    setOnPageMountMediaStreamMutex(mu)
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
    // @ts-ignore
    <MediaStreamContext.Provider value={{ mediaStream, mediaStreamReady, startNormal, startFaceDetection, onPageMountMediaStreamMutex }}>
      {children}
    </MediaStreamContext.Provider>
  )
}

export default MediaStreamProvider;
