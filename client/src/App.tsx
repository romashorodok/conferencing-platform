import { Dispatch, PropsWithChildren, Reducer, useCallback, useContext, useEffect, useReducer, useRef, useState } from 'react'
import reactLogo from './assets/react.svg'
import viteLogo from '/vite.svg'
import cameraSvg from './assets/camera.svg'
import microphoneSvg from './assets/microphone.svg'
import infoSvg from './assets/info.svg'
import './App.css'
import { isChromiumBased } from './helpers'
import { SignalContext } from './rtc/SignalProvider'
import { EventEmitter } from 'events';
import { SubscriberContext } from './rtc/SubscriberProvider'
import { MediaStreamContext, setAudioMute, setVideoMute } from './rtc/MediaStreamProvider'
import * as Popover from '@radix-ui/react-popover';
import * as ScrollArea from '@radix-ui/react-scroll-area';
import LoadingDots from './base/loading-dots'
import AppLayout from './AppLayout'

export class Mutex {
  wait: Promise<void>;
  private _locks: number;

  constructor() {
    this.wait = Promise.resolve();
    this._locks = 0;
  }

  isLocked() {
    return this._locks > 0;
  }

  lock() {
    this._locks += 1;
    let unlockNext: () => void;
    const willLock = new Promise<void>(
      (resolve) =>
      (unlockNext = () => {
        this._locks -= 1;
        resolve();
      }),
    );
    const willUnlock = this.wait.then(() => unlockNext);
    this.wait = this.wait.then(() => willLock);
    return willUnlock;
  }
}

interface RTCConfiguration {
  bundlePolicy?: RTCBundlePolicy;
  certificates?: RTCCertificate[];
  iceCandidatePoolSize?: number;
  iceServers?: RTCIceServer[];
  iceTransportPolicy?: RTCIceTransportPolicy;
  rtcpMuxPolicy?: RTCRtcpMuxPolicy;
}

enum PeerConnectionEvent {
  NegotiationNeeded = 'onnegotiationneeded',
  OnICECandidate = 'onicecandidate'
}

export class RTCEngine extends EventEmitter {
  peerConnection: RTCPeerConnection | null

  constructor(
    private readonly config?: RTCConfiguration,
    private readonly mediaConstraints: Record<string, unknown> = {}
  ) {
    super()
    this.peerConnection = this.newPeerConnection()
  }

  private newPeerConnection(): RTCPeerConnection {
    const pc = isChromiumBased()
      ? // @ts-expect-error chrome allows additional media constraints to be passed into the RTCPeerConnection constructor
      new RTCPeerConnection(this.RTCConfiguration(), this.mediaConstraints)
      : new RTCPeerConnection(this.RTCConfiguration());
    pc.onnegotiationneeded = (evt) => this.emit(PeerConnectionEvent.NegotiationNeeded, evt)
    pc.onicecandidate = (evt) => this.emit(PeerConnectionEvent.OnICECandidate, evt)
    return pc;
  }

  private RTCConfiguration(): RTCConfiguration {
    return {
      ...this.config,
      // @ts-ignore
      continualGatheringPolicy: 'gather_continually'
    };
  }

  close() {
    this.peerConnection?.close()
  }

  // Look at how they modify SDP to achieve specific codec options supported by the browser.
  // https://github.com/livekit/client-sdk-js/blob/761711adb4195dc49a0b32e1e4f88726659dac94/src/room/PCTransport.ts#L129
  async setRemoteDescription(sd: RTCSessionDescription) {
    // when remote description is an offer, my remote server is polite
    // By that the server can initiate renegotiation(change tracks, ...other params) 
    if (sd.type === 'offer') {
      this.peerConnection?.setRemoteDescription(sd)
      return
    }

    if (sd.type === 'answer') { }
    throw new Error("undefined sdp type")
  }

  async setLocalDescription(sd: RTCSessionDescriptionInit) {
    if (sd.type === 'answer') {
      this.peerConnection?.setLocalDescription(sd)
      return
    }

    if (sd.type === 'offer') { }
    throw new Error("undefined sdp type")
  }

  createAnswer() {
    return this.peerConnection?.createAnswer(undefined)
  }

  async onTrickleIceCandidate(ice: string) {
    const candidate: RTCIceCandidateInit = new RTCIceCandidate(JSON.parse(ice))
    return this.peerConnection?.addIceCandidate(candidate)
  }
}

enum SignalEvent {
  Offer = "offer",
  Answer = "answer",
  // https://datatracker.ietf.org/doc/html/draft-ietf-mmusic-trickle-ice
  // TrickleIceCandidate = "trickle-ice-candidate"
  TrickleIceCandidate = "candidate"
}

export class Signal extends EventEmitter {
  ws?: WebSocket

  public onConnect = new Mutex()
  private onConnectUnlock: Promise<() => void>
  // private lock = new Mutex()

  constructor(private readonly uri: string = "ws://localhost:8080/ws-rtc-signal") {
    super()
    this.onConnectUnlock = this.onConnect.lock()
  }

  connect() {
    this.ws = new WebSocket(this.uri);
    this.ws.onopen = async () => (await this.onConnectUnlock)()
    this.ws.onmessage = (ev) => {
      // TODO: use protobuf. Now json is good approach
      const { event = null, data = null } = JSON.parse(ev.data)
      if (!event) {
        return
      }
      this.emit(event, data)
    }
  }

  async destroy() {
    (await this.onConnectUnlock)()
    this.ws?.close()
    this.onConnect = new Mutex()
    this.onConnectUnlock = this.onConnect.lock()
  }

  async subscribe() {
    this.ws?.send(JSON.stringify({
      event: "subscribe",
      data: ""
    }))
  }

  async answer(sd: RTCSessionDescriptionInit) {
    this.ws?.send(JSON.stringify({
      event: "answer",
      data: JSON.stringify(sd)
    }))
  }
}

function useSubscriber({
  setMediaStream
}: {
  setMediaStream: Dispatch<OriginTrackEvent>
}) {
  const { subscriber } = useContext(SubscriberContext)
  const { signal } = useContext(SignalContext)

  useEffect(() => {
    if (!signal || !subscriber) return
    signal.on(SignalEvent.TrickleIceCandidate, subscriber.onTrickleIceCandidate)
    subscriber.on(PeerConnectionEvent.OnICECandidate, function(e) {
      if (!e.candidate) {
        return
      }

      console.log("send ice candidate", e.candidate)
      signal.ws?.send(JSON.stringify({
        event: SignalEvent.TrickleIceCandidate,
        data: JSON.stringify(e.candidate)
      }))
    })
    return () => {
      signal.removeAllListeners(SignalEvent.TrickleIceCandidate)
      subscriber.removeAllListeners(PeerConnectionEvent.OnICECandidate)
    }
  }, [subscriber, signal])

  useEffect(() => {
    if (!subscriber || !signal)
      return

    const onOffer = async function(sd: string) {
      if (!subscriber)
        return
      await subscriber.setRemoteDescription(JSON.parse(sd))
      let answer = await subscriber.createAnswer()
      if (!answer || !answer.sdp) {
        throw new Error("undefined subscriber an answer, when the negotiate")
      }
      answer.sdp = answer.sdp.replace("useinbandfec=1", "useinbandfec=1;stereo=1")
      subscriber.setLocalDescription(answer)
      signal.answer(answer)
    }

    signal.on(SignalEvent.Offer, onOffer)
    return () => {
      signal.removeListener(SignalEvent.Offer, onOffer)
    }
  }, [signal, subscriber])

  // process when two peers exchange session descriptions about their capabilities and establish a connection
  //
  // |> offer: server
  // |> setlocaldescription: server  
  // |> (-> (signal) clientrecv) // client receive the offer from server
  // |> setremotedescription: client
  // |> answer: client
  // |> setlocaldescripton: client
  // |> (-> (signal) serverrecv) // server receive the answer from client
  // |> setremotedescription: server
  //
  // After that, peers must send their ice candidates
  const subscribe = useCallback(async () => {
    await signal.onConnect.wait
    signal.subscribe()
  }, [signal])

  useEffect(() => {
    subscriber.peerConnection!.ontrack = (evt) => {
      // if (evt.track.kind === 'audio') {
      //   return
      // }

      // @ts-ignore
      const el: HTMLVideoElement = document.createElement(evt.track.kind)
      const [stream] = evt.streams
      setMediaStream({ [stream.id]: evt })
      // setMediaStream({ [stream.id]: stream })
      stream.onremovetrack = () => setMediaStream({ [stream.id]: undefined })
    }
  }, [])

  return {
    subscribe
  }
}

function usePublisher() {
  const { subscriber } = useContext(SubscriberContext)
  const { mediaStream } = useContext(MediaStreamContext)

  const publish = useCallback(async () => {
    try {
      const stream = mediaStream
      stream.getTracks().forEach(t => {
        subscriber.peerConnection?.addTrack(t, stream)
      })
    } catch { }
  }, [subscriber, mediaStream])

  useEffect(() => {
    const videoSenders = subscriber.peerConnection?.getSenders().filter(t => t.track?.kind === 'video')
    const audioSenders = subscriber.peerConnection?.getSenders().filter(t => t.track?.kind === 'audio')

    videoSenders?.forEach(s => {
      mediaStream.getVideoTracks().forEach(t => {
        s.replaceTrack(t)
      })
    })

    audioSenders?.forEach(s => {
      mediaStream.getAudioTracks().forEach(t => {
        s.replaceTrack(t)
      })
    })

    return () => {
      subscriber?.peerConnection?.getSenders().forEach(s => {
        s.replaceTrack(null)
        // subscriber.peerConnection?.removeTrack(s)
      })
    }
  }, [mediaStream, subscriber])

  return { publish }
}

function useRoom() {
  const [mediaStreamList, setMediaStream] = useReducer<Reducer<MediaStreamContextReducer, OriginTrackEvent>>(
    (state, event) => {
      Object.entries(event).forEach(([streamID, trackEvent]) => {
        if (!trackEvent) {
          delete state[streamID]
          return
        }

        let context: StreamContext
        if (state[streamID]) {
          context = state[streamID]
        } else {
          context = { stream: new MediaStream(), originList: [] }
        }

        context.originList.push(trackEvent)
        context.stream.addTrack(trackEvent.track)

        Object.assign(state, { [streamID]: context })
      })

      return { ...state }
    },
    {}
  )
  const { subscribe } = useSubscriber({ setMediaStream })
  const { publish } = usePublisher()

  const join = useCallback(() => publish().then(_ => subscribe()), [publish, subscribe])

  useEffect(() => console.log(mediaStreamList), [mediaStreamList])

  return { join, mediaStreamList }
}

function CameraComponent() {
  const {
    mediaStream,
    mediaStreamReady,
  } = useContext(MediaStreamContext)
  const [isVideoMuted, setVideoMuted] = useState<boolean>(mediaStream?.getVideoTracks()[0].enabled || true);
  const [isAudioMuted, setAudioMuted] = useState<boolean>(mediaStream?.getAudioTracks()[0].enabled || true);

  const video = useRef<HTMLVideoElement>(null);

  useEffect(() => {
    if (mediaStream)
      setVideoMute(mediaStream, isVideoMuted)
  }, [isVideoMuted, mediaStream])

  useEffect(() => {
    if (mediaStream)
      setAudioMute(mediaStream, isAudioMuted)
  }, [isAudioMuted, mediaStream])

  useEffect(() => {
    if (mediaStream) {
      video.current!.srcObject = mediaStream
    }
  }, [video, mediaStream])

  return (
    <div className={`p-4`}>
      <div className={`flex relative w-[448px] h-[252px]`} >
        <div className={`${mediaStreamReady ? 'visible' : 'invisible'} z-10`}>
          <video ref={video} className={`w-[448px] h-[252px]`} autoPlay muted />
          <div className={`absolute left-[41%] bottom-[10%] flex gap-6`}>
            <button className={`${isAudioMuted ? 'bg-red-500' : 'bg-green-500'} p-1 rounded-lg active:outline-none hover:outline-none focus:outline-none`} onClick={() => setAudioMuted(!isAudioMuted)}>
              <img className={`w-[24px] h-[24px]`} src={microphoneSvg} />
            </button>
            <button className={`${isVideoMuted ? 'bg-red-500' : 'bg-green-500'} p-1 rounded-lg active:outline-none hover:outline-none focus:outline-none`} onClick={() => setVideoMuted(!isVideoMuted)}>
              <img className={`w-[24px] h-[24px]`} src={cameraSvg} />
            </button>
          </div>
        </div>
        {mediaStreamReady
          ? null
          : (
            <div className={`absolute left-[43%] bottom-[10%] z-10`} >
              <p>Loading</p>
              <LoadingDots absolute={false} />
            </div>
          )}
        <span className={`z-0 absolute bg-black w-full h-full`}></span>
      </div>
    </div>
  );
};

function RoomParticipant({
  mediaStream,
  isLoading,
}: {
  mediaStream: MediaStream,
  isLoading: boolean,
}) {
  const video = useRef<HTMLVideoElement>(null)

  useEffect(() => {
    if (video.current) {
      const vi = video.current
      vi.srcObject = mediaStream
      vi.autoplay = true
    }
  }, [video, mediaStream])

  useEffect(() => {
    if (!mediaStream || !video.current) return
    const vi = video.current

    const [videoTrack] = mediaStream.getVideoTracks()
    if (!videoTrack) return

    videoTrack.onmute = function() {
      vi.play()
    }
  }, [video, mediaStream])

  return (
    <div>
      <div className={`flex relative w-[448px] h-[252px]`} >
        <div className={`${isLoading ? 'invisible' : 'visible'} z-10`}>
          <video ref={video} className={`w-[448px] h-[252px]`} />
        </div>
        {isLoading
          ? (
            <div className={`absolute left-[43%] bottom-[10%] z-10`} >
              <p>Loading</p>
              <LoadingDots absolute={false} />
            </div>
          )
          : null}
        <span className={`z-0 absolute bg-black w-full h-full`}></span>
      </div>
    </div>
  )
}

enum StatType {
  Codec = "codec",
  Inbound = "inbound-rtp",
}

enum TrackKind {
  Video = "video",
  Audio = "audio",
}

type StatObject = {
  id: string,
  type: string,
  kind: string,
  mimeType: string,
  // At least 30 frames
  framesDecoded: number,
  framesReceived: number,
}

function StatByKind(statList: StatObject[]): Map<string, StatObject> {
  const result = new Map<string, StatObject>();

  for (const kindName of Object.values(TrackKind)) {
    let kindStat: StatObject = {} as StatObject;
    for (const stat of statList) {
      if (stat.kind !== kindName && !stat.mimeType?.includes(kindName)) {
        continue;
      }
      kindStat = { ...stat, ...kindStat }
    }
    result.set(kindName, kindStat);
  }
  return result;
}

function StreamStats({
  streamID,
  statList,
}: {
  streamID: string,
  statList: Map<string, StatObject>,
}) {
  return (
    <div>
      <p>Stream: {streamID}</p>
      {Array.from(statList).map(([kind, stats]) => (
        <div key={kind}>
          <br />
          <p>Kind: {kind}</p>
          <br />
          <table>
            <thead>
              <tr>
                <th>Name</th>
                <th>Value</th>
              </tr>
            </thead>
            <tbody>
              {Object.entries(stats).map(([name, val]) =>
                <tr key={name}>
                  <td>{name}</td>
                  <td>{val}</td>
                </tr>
              )}
            </tbody>
          </table>
        </div>
      ))}
    </div>
  )
}

function RoomStatContainer({
  streamID,
  statList,
  children,
  show
}: PropsWithChildren<{
  streamID: string,
  statList: Map<string, StatObject>,
  show: boolean
}>) {
  return (
    <div className={`absolute top-[0] left-[0] z-30`}>
      <Popover.Root open={show}>
        <Popover.Trigger asChild>
          {children}
        </Popover.Trigger>
        <Popover.Portal>
          <Popover.Content className={`z-30`}>
            <ScrollArea.Root className={`flex p-4 w-[448px] h-[252px] bg-black-t border-dimgray rounded z-30`}>
              <ScrollArea.Viewport>
                <StreamStats statList={statList} streamID={streamID} />
              </ScrollArea.Viewport>
              <ScrollArea.Scrollbar orientation="vertical">
                <ScrollArea.Thumb />
              </ScrollArea.Scrollbar>
              <ScrollArea.Scrollbar orientation="horizontal">
                <ScrollArea.Thumb />
              </ScrollArea.Scrollbar>
              <ScrollArea.Corner />
            </ScrollArea.Root>
          </Popover.Content>
        </Popover.Portal>
      </Popover.Root>
    </div>
  )
}

const FRAMES_DECODED_STOP_LOADING = 1

function RoomStream({
  mediaStream,
}: { mediaStream: MediaStream }) {
  const { subscriber } = useContext(SubscriberContext)
  const [statList, setStatList] = useState(new Map<string, StatObject>())
  const [showStats, setShowStats] = useState(false)
  const [isLoading, setIsLoading] = useState(true)

  const getStats = useCallback(async () => {
    const tracks = mediaStream.getTracks()
    const stats: StatObject[] = []
    await Promise.all(tracks.map(async track => {
      const report = await subscriber.peerConnection?.getStats(track);
      if (report) {
        for (const [_, stat] of Array.from(report)) {
          switch (stat.type) {
            case StatType.Codec:
            case StatType.Inbound:
              stats.push(stat);
              break;
            default:
          }
        }
      }
    }));
    setStatList(StatByKind(stats))
  }, [subscriber, mediaStream])

  useEffect(() => {
    const intervalId = setInterval(() => getStats(), 500)
    return () => clearInterval(intervalId)
  }, [getStats])

  useEffect(() => {
    if (!isLoading) return

    const decodedFramesCount = statList.get(TrackKind.Video)?.framesDecoded
    if (decodedFramesCount && decodedFramesCount >= FRAMES_DECODED_STOP_LOADING) {
      setIsLoading(false)
    }
  }, [statList])

  return (
    <div className={`relative`}>
      <RoomParticipant mediaStream={mediaStream} isLoading={isLoading} />
      <RoomStatContainer show={showStats} streamID={mediaStream.id} statList={statList} >
        <button onClick={() => setShowStats(!showStats)}>
          <img className={`w-[22px] h-[22px] filter-white`} src={infoSvg} />
        </button>
      </RoomStatContainer>
    </div>
  )
}

function Room() {
  const { join, mediaStreamList } = useRoom()

  return (
    <div>
      <button onClick={join}>Join</button>
      {Object.entries(mediaStreamList).map(([id, { stream }]) => (
        <RoomStream key={id} mediaStream={stream} />
      ))}
    </div>
  )
}

type StreamContext = {
  stream: MediaStream,
  originList: Array<RTCTrackEvent>
}

type MediaStreamContextReducer = { [streamID: string]: StreamContext }

type DeleteStreamEvent = undefined
type OriginTrackEvent = { [streamID: string]: RTCTrackEvent | DeleteStreamEvent }

function App() {
  const { startFaceDetection, startNormal } = useContext(MediaStreamContext)

  // const { join } = useRoom()

  const [count, setCount] = useState(0)

  useEffect(() => console.log(count), [count])

  return (
    <AppLayout>
      <button onClick={startNormal}>Normal</button>
      <button onClick={startFaceDetection}>Face detection</button>

      <CameraComponent />
      <Room />

      <img id="image" src={reactLogo} className="logo react" alt="React logo" />
      <div>
        <h3> Remote Video </h3>
        <div id="remoteVideos"></div> <br />

        <a href="https://vitejs.dev" target="_blank">
          <img src={viteLogo} className="logo" alt="Vite logo" />
        </a>
        <a href="https://react.dev" target="_blank">
          <img src={reactLogo} className="logo react" alt="React logo" />
        </a>
      </div>
      <h1>Vite + React</h1>
      <div className="card">
        <button onClick={() => setCount((count) => count + 1)}>
          count is {count}
        </button>
        <p>
          Edit <code>src/App.tsx</code> and save to test HMR
        </p>
      </div>
      <p className="read-the-docs">
        Click on the Vite and React logos to learn more
      </p>

    </AppLayout>
  )
}

export default App
