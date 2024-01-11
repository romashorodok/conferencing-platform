import { useCallback, useContext, useEffect, useLayoutEffect, useState } from 'react'
import reactLogo from './assets/react.svg'
import viteLogo from '/vite.svg'
import './App.css'
import { isChromiumBased } from './helpers'
import { SignalContext, useSignal } from './rtc/SignalProvider'
import { EventEmitter } from 'events';
import { SubscriberContext } from './rtc/SubscriberProvider'

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

export class RTCEngine {
  peerConnection: RTCPeerConnection | null

  constructor(
    private readonly config?: RTCConfiguration,
    private readonly mediaConstraints: Record<string, unknown> = {}
  ) {
    this.peerConnection = this.newPeerConnection()
  }

  private newPeerConnection(): RTCPeerConnection {
    const pc = isChromiumBased()
      ? // @ts-expect-error chrome allows additional media constraints to be passed into the RTCPeerConnection constructor
      new RTCPeerConnection(this.RTCConfiguration(), this.mediaConstraints)
      : new RTCPeerConnection(this.RTCConfiguration());
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

  addIceCandidate(candidate: RTCIceCandidateInit) {
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
  private lock = new Mutex()

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

// const googConstraints = { optional: [{ googDscp: true }] };
// function usePublisher() {
//   const publisher = useState(new RTCEngine({}, googConstraints))
// }

function useSubscriber() {
  const { subscriber } = useContext(SubscriberContext)
  const { signal } = useContext(SignalContext)

  // Process when two peers exchange session descriptions about their capabilities and establish a connection
  //
  // |> offer: server
  // |> setLocalDescription: server  
  // |> (-> (Signal) clientRecv) // Client receive the offer from server
  // |> setRemoteDescription: client
  // |> answer: client
  // |> setLocalDescripton: client
  // |> (-> (Signal) serverRecv) // Server receive the answer from client
  // |> setRemoteDescription: server
  //
  const negotiation = useCallback(async (sd: RTCSessionDescription) => {
    signal.on(SignalEvent.TrickleIceCandidate, function onTrickleIceCandidate(iceCandidate) {
      try {
        subscriber.addIceCandidate(new RTCIceCandidate(JSON.parse(iceCandidate)))
      } finally {
        signal.off(SignalEvent.TrickleIceCandidate, onTrickleIceCandidate)
      }
    })

    try {
      await subscriber.setRemoteDescription(sd)
      const answer = await subscriber.createAnswer()
      if (!answer) {
        throw new Error("undefined subscriber an answer, when the negotiate")
      }
      await subscriber.setLocalDescription(answer)
      await signal.answer(answer)
    } catch (err) {
      console.error(err)
    }
  }, [subscriber, signal])

  const subscribe = useCallback(async () => {
    await signal.onConnect.wait
    if (subscriber.peerConnection?.connectionState !== 'connected') {
      signal.subscribe()
    }
  }, [signal, negotiation])

  useEffect(() => {
    const onOffer = (offer: any) => {
      try {
        console.log("offer", JSON.parse(offer))
        negotiation(JSON.parse(offer))
      } catch (err) {
        console.log(err)
      }
      // finally {
      //   signal.off(SignalEvent.Offer, onOffer)
      // }
    }
    signal.on(SignalEvent.Offer, onOffer)
    return () => {
      signal.off(SignalEvent.Offer, onOffer)
    }
  }, [signal, negotiation])

  useEffect(() => {
    (async () => {


      if (subscriber?.peerConnection) {
        const stream = await navigator.mediaDevices.getUserMedia({ video: true, audio: true })
        stream.getTracks().forEach(track => subscriber.peerConnection?.addTrack(track, stream))

        subscriber.peerConnection.ontrack = (event) => {
          if (event.track.kind === 'audio') {
            return
          }

          let el: HTMLVideoElement = document.createElement(event.track.kind)
          el.srcObject = event.streams[0]
          el.autoplay = true
          el.controls = true
          document.getElementById('remoteVideos').appendChild(el)

          event.track.onmute = function(event) {
            el.play()
          }

          event.streams[0].onremovetrack = ({ track }) => {
            if (el.parentNode) {
              el.parentNode.removeChild(el)
            }
          }

        }
      }
    }
    )()
  }, [])

  return {
    subscribe
  }
}

// type IngestResponse = {
//   answer: {
//     sdp: string
//     type: string
//   }
// };

// function useWebrtcDataChannelStats() {
//   const dataChannelLabel = "signaling";
//
//   function startStatDataChannel(peerConnection: RTCPeerConnection): RTCDataChannel {
//     const dataChannel = peerConnection.createDataChannel(dataChannelLabel, {})
//
//     return dataChannel
//   }
//
//   return {
//     startStatDataChannel
//   }
// }

// function useWebrtcIngest() {
//   const { peerConnection, setPeerConnection } = useContext(PeerConnectionContext)
//
//   const { startStatDataChannel } = useWebrtcDataChannelStats()
//
//   const ingest = useCallback(async () => {
//     try {
//       const statDataChannel = startStatDataChannel(peerConnection)
//
//       statDataChannel.onmessage = async (dc: MessageEvent<any>) => {
//         try {
//           const { type, sdp } = JSON.parse(dc.data)
//           //
//           // console.log("answer", sdp)
//
//           //
//           const offer = new RTCSessionDescription({
//             type: 'pranswer'
//           })
//           await peerConnection.setLocalDescription(offer)
//
//           // await peerConnection.setRemoteDescription(answer)
//
//           // const offer = await peerConnection.createOffer({ iceRestart: false })
//
//           // console.log("offer", offer)
//           // const offerJSON = JSON.stringify(offer)
//           // console.log("offer json", offerJSON)
//
//           // statDataChannel.send(JSON.stringify(answer))
//
//           await peerConnection.setRemoteDescription(
//             new RTCSessionDescription({
//               type,
//               sdp
//             })
//           );
//
//           // const answer = await peerConnection.createAnswer()
//           //
//           // console.log("Sucess send answer to peer", answer)
//           //
//           // statDataChannel.send(JSON.stringify(answer))
//
//         } catch (err) {
//           console.log(err)
//         }
//       }
//
//       const offer = await peerConnection.createOffer()
//
//       await peerConnection.setLocalDescription(offer)
//
//       const response: IngestResponse = await fetch('http://localhost:8080/ingress/whip/session/test', {
//         method: 'POST',
//         body: JSON.stringify({
//           offer: {
//             sdp: offer.sdp,
//             type: offer.type,
//           }
//         })
//       }).then(r => r.json())
//
//       const desc = new RTCSessionDescription({
//         type: 'answer',
//         sdp: response.answer.sdp,
//       })
//
//       await peerConnection.setRemoteDescription(desc)
//
//     } catch (err) {
//       console.log(err)
//     }
//   }, [peerConnection])
//
//   const getInfo = useCallback(() => {
//     console.log("senders", peerConnection.getSenders())
//     console.log("receivers", peerConnection.getReceivers())
//     console.log("transceivers", peerConnection.getTransceivers())
//   }, [peerConnection])
//
//   const stop = useCallback(() => {
//     peerConnection.close()
//     setPeerConnection(undefined as never)
//   }, [peerConnection]);
//
//   return {
//     getInfo,
//     ingest,
//     stop
//   }
// }

// function Broadcast() {
//   const { peerConnection } = useContext(PeerConnectionContext)
//
//   function Broadcast() {
//     navigator.mediaDevices.getUserMedia({ video: true, audio: true })
//       .then(stream => {
//         const pc = peerConnection
//         pc.ontrack = function(event) {
//           if (event.track.kind === 'audio') {
//             return
//           }
//
//           console.log(event.track)
//
//           let el: HTMLVideoElement = document.createElement(event.track.kind)
//           el.srcObject = event.streams[0]
//           el.autoplay = true
//           el.controls = true
//           document.getElementById('remoteVideos').appendChild(el)
//
//           event.track.onmute = function(event) {
//             el.play()
//           }
//
//           event.streams[0].onremovetrack = ({ track }) => {
//             if (el.parentNode) {
//               el.parentNode.removeChild(el)
//             }
//           }
//         }
//
//         document.getElementById('localVideo').srcObject = stream
//
//         stream.getTracks().forEach(track => pc.addTrack(track, stream))
//
//       }).catch(window.alert)
//
//   }
//
//
//   return (
//     <>
//       <button onClick={Broadcast}>Broadcast</button>
//       <h3> Local Video </h3>
//       <video id="localVideo" width="160" height="120" autoplay muted></video> <br />
//
//       <h3> Remote Video </h3>
//       <div id="remoteVideos"></div> <br />
//
//       <h3> Logs </h3>
//       <div id="logs"></div>
//
//     </>
//   )
// }

function Main() {
  // const { ingest, stop, getInfo } = useWebrtcIngest()

  return (
    <>
      <button onClick={stop}>Stop</button>
    </>
  )
}


function App() {
  const { subscribe } = useSubscriber()

  const [count, setCount] = useState(0)

  useEffect(() => console.log(count), [count])

  return (
    <>
      <Main />
      <div>
        <h3> Remote Video </h3>
        <div id="remoteVideos"></div> <br />


        <button onClick={subscribe}>Subscribe</button>

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
