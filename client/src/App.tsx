import { Dispatch, Reducer, useCallback, useContext, useEffect, useReducer, useRef, useState } from 'react'
import reactLogo from './assets/react.svg'
import viteLogo from '/vite.svg'
import cameraSvg from './assets/camera.svg'
import microphoneSvg from './assets/microphone.svg'
import './App.css'
import { isChromiumBased } from './helpers'
import { SignalContext } from './rtc/SignalProvider'
import { EventEmitter } from 'events';
import { SubscriberContext } from './rtc/SubscriberProvider'
import { MediaStreamContext } from './rtc/MediaStreamProvider'

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
  setMediaStream: Dispatch<SetMediaStreamAction>
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
      const answer = await subscriber.createAnswer()
      if (!answer) {
        throw new Error("undefined subscriber an answer, when the negotiate")
      }
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

    // if (subscriber.peerConnection?.connectionState !== 'connected') {
    signal.subscribe()
    // }
  }, [signal])


  useEffect(() => {
    subscriber.peerConnection!.ontrack = (evt) => {
      if (evt.track.kind === 'audio') {
        return
      }

      // @ts-ignore
      const el: HTMLVideoElement = document.createElement(evt.track.kind)
      const [stream] = evt.streams

      setMediaStream({ [stream.id]: stream })

      stream.onremovetrack = () => setMediaStream({ [stream.id]: undefined })

      // el.srcObject = event.streams[0]
      // event.streams[0].addEventListener('removetrack', function(evt) {
      //   console.log("remove track stream end", evt)
      // })
      //
      // el.autoplay = true
      // el.controls = true
      // document.getElementById('remoteVideos')!.appendChild(el)
      //
      // event.track.onmute = function() {
      //   el.play()
      // }
      //
      // event.track.onended = function(evt) {
      //   console.log("Stream ended", evt)
      // }
      //
      // event.streams[0].onremovetrack = (evt) => {
      //   console.log("stream", evt)
      //   if (el.parentNode) {
      //     el.parentNode.removeChild(el)
      //   }
      // }

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
      const stream = await mediaStream
      stream.getTracks().forEach(t => {
        subscriber.peerConnection?.addTrack(t, stream)
      })
    } catch { }
  }, [subscriber, mediaStream])

  useEffect(() => () => {
    subscriber?.peerConnection?.getSenders().forEach(s => {
      subscriber.peerConnection?.removeTrack(s)
    })
  }, [mediaStream, subscriber])

  return { publish }
}

function useRoom() {
  const [mediaStreamList, setMediaStream] = useReducer<Reducer<MediaStreamReducerState, SetMediaStreamAction>>(
    (state, action) => {
      for (const [streamID, mediaStream] of Object.entries(action)) {
        if (!mediaStream) {
          delete state[streamID]
          continue
        }
        Object.assign(state, { [streamID]: mediaStream })
      }
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
  const { mediaStream, mediaStreamReady } = useContext(MediaStreamContext)
  const [isVideoMuted, setVideoMuted] = useState(true);
  const [isAudioMuted, setAudioMuted] = useState(true);

  const video = useRef<HTMLVideoElement>(null);

  useEffect(() => {
    async function changeVideoMute() {
      const stream = await mediaStream
      stream.getVideoTracks().forEach(t => t.enabled = !isVideoMuted)
    }
    changeVideoMute()
  }, [isVideoMuted, mediaStream])

  useEffect(() => {
    async function changeAudioMute() {
      const stream = await mediaStream
      stream.getAudioTracks().forEach(t => t.enabled = !isVideoMuted)
    }
    changeAudioMute()
  }, [isAudioMuted, mediaStream])

  useEffect(() => {
    mediaStream.then(s => { video.current!.srcObject = s })
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
        {mediaStreamReady ? null : (<p className={`absolute left-[43%] bottom-[10%] z-10`}>Loading...</p>)}
        <span className={`z-0 absolute bg-black w-full h-full`}></span>
      </div>
    </div>
  );
};

function RoomParticipant({
  mediaStream
}: {
  mediaStream: MediaStream
}) {
  const video = useRef<HTMLVideoElement>(null)

  useEffect(() => {
    if (video.current) {
      const vi = video.current
      vi.srcObject = mediaStream
      vi.autoplay = true
      vi.controls = true
    }
  }, [video, mediaStream])

  useEffect(() => {
    if (!mediaStream || !video.current) return
    const vi = video.current

    const [track] = mediaStream.getVideoTracks()
    track.onmute = function() {
      vi.play()
    }
  }, [video, mediaStream])

  return (
    <div>
      <video ref={video} />
    </div>
  )
}


enum StatType {
  Codec = "codec",
  Inbound = "inbound-rtp",
}

enum StatKind {
  Video = "video",
  Audio = "audio",
}

type StatObject = {
  id: string,
  type: string,
  kind: string,
  mimeType: string
}

function StatByKind(statList: StatObject[]): Map<string, StatObject> {
  const result = new Map<string, StatObject>();

  for (const kindName of Object.values(StatKind)) {
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
  mediaStream
}: {
  mediaStream: MediaStream
}) {
  const { subscriber } = useContext(SubscriberContext)
  const [statList, setStatList] = useState(new Map<string, StatObject>())

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
    const intervalId = setInterval(() => getStats(), 1000)
    return () => {
      clearInterval(intervalId)
    }
  }, [getStats])

  return (
    <div>
      <p>Stats - {mediaStream.id}</p>
      {Array.from(statList).map(([kind, stats]) => (
        <table key={kind}>
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
      ))}
    </div>
  )
}

function Room() {
  const { join, mediaStreamList } = useRoom()

  return (
    <div>
      <button onClick={join}>Join</button>

      {Object.entries(mediaStreamList).map(([id, mediaStream]) => (
        <div key={id} className={`relative`}>
          <RoomParticipant mediaStream={mediaStream} />
          <div className={`absolute top-[0] right-[0]`}>
            <StreamStats mediaStream={mediaStream} />
          </div>
        </div>
      ))}
    </div>
  )
}

type MediaStreamReducerState = { [streamID: string]: MediaStream }

type SetMediaStreamAction = { [streamID: string]: MediaStream | undefined }

function App() {


  // const { join } = useRoom()

  const [count, setCount] = useState(0)

  useEffect(() => console.log(count), [count])

  return (
    <>
      <CameraComponent />
      <Room />

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

    </>
  )
}

export default App
