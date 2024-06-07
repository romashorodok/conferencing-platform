import { PropsWithChildren, useCallback, useContext, useEffect, useRef, useState } from 'react'
import cameraSvg from './assets/camera.svg'
import microphoneSvg from './assets/microphone.svg'
import infoSvg from './assets/info.svg'
import './App.css'
import { isChromiumBased } from './helpers'
import { SignalContext } from './rtc/SignalProvider'
import { EventEmitter } from 'events';
import { SubscriberContext } from './rtc/SubscriberProvider'
import { MediaStreamContext, } from './rtc/MediaStreamProvider'
import * as Popover from '@radix-ui/react-popover';
import * as ScrollArea from '@radix-ui/react-scroll-area';
import LoadingDots from './base/loading-dots'
import { MEDIA_SERVER_STUN, MEDIA_SERVER_WS } from './variables'
import { MediaStreamContextReducer, RoomMediaStreamListContext } from './rtc/RoomMediaStreamListProvider'
import { ControlsContext } from './rtc/ControlsProvider'

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
    if (this.peerConnection?.signalingState !== 'closed') {
      this.peerConnection?.getSenders().forEach(s => this.peerConnection?.removeTrack(s))
    }

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

const attachTracks = (peerContext: RTCEngine, tracks: MediaStreamTrack[], mediaStream: MediaStream) => {
  tracks.forEach(t => {
    peerContext.peerConnection?.addTrack(t, mediaStream)
  })
}

const replaceTrack = (senders: RTCRtpSender[] | undefined, tracks: MediaStreamTrack[]) => {
  if (!senders) {
    throw Error("empty senders")
  }
  senders.forEach(s => {
    tracks.forEach(t => {
      s.replaceTrack(t)
    })
  })
}

export enum SignalEvent {
  Offer = "offer",
  Answer = "answer",
  // https://datatracker.ietf.org/doc/html/draft-ietf-mmusic-trickle-ice
  // TrickleIceCandidate = "trickle-ice-candidate"
  TrickleIceCandidate = "candidate",

  Filters = "filters"
}

export class Signal extends EventEmitter {
  ws?: WebSocket

  public onConnect = new Mutex()
  private onConnectUnlock: Promise<() => void>
  // private lock = new Mutex()

  constructor(public readonly uri: string = "ws://localhost:8080/ws-rtc-signal") {
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

  async commitOfferState(state_hash: string) {
    this.ws?.send(JSON.stringify({
      event: "commit-offer-state",
      data: JSON.stringify({
        state_hash,
      }),
    }))
  }
}

export type Filter = { name: string, mimeType: string, enabled: boolean }
type FilterMessage = { audio: Array<Filter>, video: Array<Filter> }

type OfferMessage = { hash_state: string } & RTCSessionDescription;

export function useRoom() {
  const { mediaStream } = useContext(MediaStreamContext)
  const { peerContext, setPeerContext } = useContext(SubscriberContext)
  const { signal, setSignal } = useContext(SignalContext)
  const [videoFilterList, setVideoFilterList] = useState<Array<Filter>>([])

  const { roomMediaStreamList, setRoomMediaStream } = useContext(RoomMediaStreamListContext)
  const [roomMediaList, setRoomMediaList] = useState<MediaStreamContextReducer>({})

  useEffect(() => {
    (async () => {
      const list = await roomMediaStreamList
      console.log(list)
      for (const [id, { stream }] of Object.entries(list)) {
        console.log("--- BEGIN", id, "BEGIN ---")
        console.log(stream.getTracks())
        console.log("--- END", id, "END ---")
      }

      setRoomMediaList(list)
    })()
  }, [roomMediaStreamList])


  const sinkMediaStream = useCallback(() => {
    if (!mediaStream || !peerContext)
      return

    const senders = peerContext.peerConnection?.getSenders()
    if (senders?.length === 0) {
      attachTracks(peerContext, mediaStream.getTracks(), mediaStream)
    } else {
      const videoSenders = peerContext?.peerConnection?.getSenders().filter(t => t.track?.kind === 'video')
      const audioSenders = peerContext?.peerConnection?.getSenders().filter(t => t.track?.kind === 'audio')
      replaceTrack(videoSenders, mediaStream.getVideoTracks())
      replaceTrack(audioSenders, mediaStream.getAudioTracks())
    }
  }, [mediaStream, peerContext])

  useEffect(() => {
    sinkMediaStream()

    return () => {
      // TODO: need remove tracks by hand
    }
  }, [sinkMediaStream])

  const isSameRoom = useCallback((roomID: string) => {
    const uri = `${MEDIA_SERVER_WS}/rooms/${roomID}`
    if (signal?.uri && signal.uri === uri) {
      return true
    }
    return false
  }, [signal])

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
  const join = useCallback(async ({ roomID }: { roomID: string }) => {
    if (isSameRoom(roomID))
      return

    const signalRoomUrl = `${MEDIA_SERVER_WS}/rooms/${roomID}`
    const signal = new Signal(signalRoomUrl)
    let peerContext: RTCEngine
    console.log("STUN", MEDIA_SERVER_STUN)
    if (MEDIA_SERVER_STUN !== undefined && MEDIA_SERVER_STUN !== "") {
      peerContext = new RTCEngine({
        iceServers: [
          {
            urls: MEDIA_SERVER_STUN,
          },
        ]
      })
    } else {
      peerContext = new RTCEngine()
    }

    try {
      signal.on(SignalEvent.Filters, (payload: string) => {
        try {
          const filterMessage: FilterMessage = JSON.parse(payload);
          console.log("got filters", filterMessage);
          setVideoFilterList(filterMessage.video)
        } catch {
          console.error("filters parsing error");
        }
      })

      signal.connect()
      await signal.onConnect.wait

      signal.on(SignalEvent.TrickleIceCandidate, (payload: string) => {
        const candidate = JSON.parse(payload) as RTCIceCandidate
        console.log("[Room ICE] Set ice candidate", candidate)
        peerContext.peerConnection?.addIceCandidate(candidate)
      })

      signal.on(SignalEvent.Offer, async (sd: string) => {
        const offer: OfferMessage = JSON.parse(sd)
        await peerContext.setRemoteDescription(offer)

        console.info("[OfferMessage] Recv offer state.", offer.hash_state)
        let answer = await peerContext.createAnswer()
        if (!answer || !answer.sdp) {
          throw new Error("undefined subscriber an answer, when the negotiate")
        }
        answer.sdp = answer.sdp.replace("useinbandfec=1", "useinbandfec=1;stereo=1")
        peerContext.setLocalDescription(answer)

        try {
          await signal.answer(answer)
          await signal.commitOfferState(offer.hash_state)
        } catch {
          console.error("[SignalEvent.Offer] Unable send and commit answer")
        }
      })

      peerContext.on(PeerConnectionEvent.OnICECandidate, (e: RTCIceCandidate) => {
        if (!e.candidate) {
          return
        }

        console.log("[Room Signal] send ice candidate", e.candidate)
        signal.ws?.send(JSON.stringify({
          event: SignalEvent.TrickleIceCandidate,
          data: JSON.stringify(e.candidate)
        }))
      })

      peerContext.peerConnection!.ontrack = (evt) => {
        console.log("on track", evt)

        const [stream] = evt.streams
        setRoomMediaStream({ action: 'mutate', payload: { [stream.id]: evt } })

        stream.onremovetrack = (evt) => {
          console.log("on remove track", evt)
          setRoomMediaStream({ action: 'remove', payload: { [stream.id]: evt } })
        }
      }

      signal.subscribe()
      setSignal(signal)
      setPeerContext(peerContext)
    } catch {
    }
  }, [])

  const setVideoFilter = useCallback(async (filter: Filter) => {
    if (!signal) return
    await signal.onConnect.wait

    signal.ws?.send(JSON.stringify({
      event: "filter",
      data: JSON.stringify(filter)
    }))

  }, [signal]);

  return { join, roomMediaList, sinkMediaStream, videoFilterList, setVideoFilter }
}

export function VideoControl({ className }: { className?: string }) {
  const { isVideoMuted, setVideoMuted } = useContext(ControlsContext)
  return (
    <button className={`${isVideoMuted ? 'bg-red-500' : 'bg-green-500'} cursor-pointer p-1 rounded-lg active:outline-none hover:outline-none focus:outline-none ${className}`} onClick={() => setVideoMuted(!isVideoMuted)}>
      <img className={`w-[24px] h-[24px]`} src={cameraSvg} />
    </button>
  )
}

export function AudioControl({ className }: { className?: string, width?: number, height?: number }) {
  const { isAudioMuted, setAudioMuted } = useContext(ControlsContext)
  return (
    <button className={`${isAudioMuted ? 'bg-red-500' : 'bg-green-500'} cursor-pointer p-1 rounded-lg active:outline-none hover:outline-none focus:outline-none ${className}`} onClick={() => setAudioMuted(!isAudioMuted)}>
      <img className={`w-[24px] h-[24px]`} src={microphoneSvg} />
    </button>
  )
}


export function VideoControlLarge({ className }: { className?: string }) {
  const { isVideoMuted, setVideoMuted } = useContext(ControlsContext)
  return (
    <button className={`${isVideoMuted ? 'bg-red-500' : 'bg-green-500'} cursor-pointer p-1 rounded-lg active:outline-none hover:outline-none focus:outline-none ${className}`} onClick={() => setVideoMuted(!isVideoMuted)}>
      <img className={`w-[32px] h-[32px]`} src={cameraSvg} />
    </button>
  )
}

export function AudioControlLarge({ className }: { className?: string, width?: number, height?: number }) {
  const { isAudioMuted, setAudioMuted } = useContext(ControlsContext)
  return (
    <button className={`${isAudioMuted ? 'bg-red-500' : 'bg-green-500'} cursor-pointer p-1 rounded-lg active:outline-none hover:outline-none focus:outline-none ${className}`} onClick={() => setAudioMuted(!isAudioMuted)}>
      <img className={`w-[32px] h-[32px]`} src={microphoneSvg} />
    </button>
  )
}

export function CameraComponent() {
  const {
    mediaStream,
    mediaStreamReady,
  } = useContext(MediaStreamContext)

  const video = useRef<HTMLVideoElement>(null);

  useEffect(() => {
    if (mediaStream) {
      video.current!.srcObject = mediaStream
    }
  }, [video, mediaStream])

  return (
    <div className={`min-w-[400px] min-h-[120px] max-w-max max-h-max`}>
      <div className={`flex place-self-center place-content-center relative w-full h-full`} >
        <div className={`${mediaStreamReady ? 'visible' : 'invisible'} contents z-10`}>
          <video ref={video} className={`z-20 w-5/6 h-5/6 object-cover place-self-center`} autoPlay muted />
          <div className={`absolute left-[41%] bottom-[10%] z-20 flex gap-6`}>
            <AudioControl />
            <VideoControl />
          </div>
        </div>
        {mediaStreamReady
          ? null
          : (
            <div className={`absolute left-[43%] bottom-[10%] z-20`} >
              <p>Loading</p>
              <LoadingDots absolute={false} />
            </div>
          )}
        <span className={`z-10 absolute bg-black w-full h-full`}></span>
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
    <div className={`flex place-self-center place-content-center relative w-full h-full`} >
      <div className={`${isLoading ? 'invisible' : 'visible'} contents`}>
        <video ref={video} className={`z-10 w-5/6 h-5/6 place-self-center`} />
      </div>
      {isLoading
        ? (
          <div className={`absolute left-[43%] bottom-[10%] z-10`} >
            <p>Loading</p>
            <LoadingDots absolute={false} />
          </div>
        )
        : null}
      {/* <span className={`z-0 absolute bg-black h-full w-full rounded-lg`}></span> */}
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
    <div className={`absolute top-[10px] right-[10px] z-30`}>
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

export function RoomStream({
  mediaStream,
}: { mediaStream: MediaStream }) {
  const { peerContext: subscriber } = useContext(SubscriberContext)
  const [statList, setStatList] = useState(new Map<string, StatObject>())
  const [showStats, setShowStats] = useState(false)
  const [isLoading, setIsLoading] = useState(true)

  const getStats = useCallback(async () => {
    const tracks = mediaStream.getTracks()
    const stats: StatObject[] = []
    await Promise.all(tracks.map(async track => {
      const report = await subscriber!.peerConnection?.getStats(track);
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
    <div className={`flex self-center relative h-full`}>
      <RoomParticipant mediaStream={mediaStream} isLoading={isLoading} />
      <RoomStatContainer show={showStats} streamID={mediaStream.id} statList={statList} >
        <button onClick={() => setShowStats(!showStats)}>
          <img className={`w-[22px] h-[22px] filter-white`} src={infoSvg} />
        </button>
      </RoomStatContainer>
    </div>
  )
}

function App() {
  const { startNormal } = useContext(MediaStreamContext)

  return (
    <div className="flex flex-col">
      <h1 className="text-lg">Test your camera or select room</h1>

      <CameraComponent />
      <button
        className="Button box-border w-[120px] inline-flex h-[35px] items-center justify-center rounded-[4px] px-[15px] font-medium leading-none cursor-pointer focus:outline-none mt-[10px]" onClick={() => startNormal()}>Normal</button>
    </div>
  )
}

export default App
