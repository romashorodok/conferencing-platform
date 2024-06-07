import { PropsWithChildren, createContext, useEffect, useState } from "react";
import { isChromiumBased, isFireFox } from "../helpers";
import { Mutex } from "../App";

type MediaStreamContextType = {
  mediaStream: MediaStream
  mediaStreamReady: boolean
  onPageMountMediaStreamMutex: Mutex | null

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
  // NOTE: Selecting video may be work only on localhost
  const inbound = navigator.mediaDevices.getUserMedia({
    video: {
      width: 320,
      height: 180,
    },
  })

  videoStream = inbound
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

function MediaStreamProvider({ children }: PropsWithChildren<{}>) {
  const [mediaStream, setMediaStream] = useState<MediaStream>()
  const [mediaStreamReady, setMediaStreamReady] = useState(false)
  const [worker, setWorker] = useState<Worker>()
  const [onPageMountMediaStreamMutex, setOnPageMountMediaStreamMutex] = useState<Mutex | null>(null)

  async function startNormal(unlock?: Promise<() => void>) {
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

  useEffect(() => {
    const mu = new Mutex()
    const unlock = mu.lock()

    if (!navigator.mediaDevices?.enumerateDevices) {
      console.log("enumerateDevices() not supported.");
    } else {
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
    <MediaStreamContext.Provider value={{ mediaStream, mediaStreamReady, startNormal, onPageMountMediaStreamMutex }}>
      {children}
    </MediaStreamContext.Provider>
  )
}

export default MediaStreamProvider;
