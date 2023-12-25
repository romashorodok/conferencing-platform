import { useCallback, useContext, useEffect, useState } from 'react'
import reactLogo from './assets/react.svg'
import viteLogo from '/vite.svg'
import './App.css'
import { PeerConnectionContext } from './PeerConnectionProvider'

type IngestResponse = {
  answer: {
    sdp: string
    type: string
  }
};

function useWebrtcDataChannelStats() {
  const dataChannelLabel = "signaling";

  function startStatDataChannel(peerConnection: RTCPeerConnection): RTCDataChannel {
    const dataChannel = peerConnection.createDataChannel(dataChannelLabel, {})

    return dataChannel
  }

  return {
    startStatDataChannel
  }
}

function useWebrtcIngest() {
  const { peerConnection, setPeerConnection } = useContext(PeerConnectionContext)

  const { startStatDataChannel } = useWebrtcDataChannelStats()

  const ingest = useCallback(async () => {
    try {
      const statDataChannel = startStatDataChannel(peerConnection)

      statDataChannel.onmessage = (dc: MessageEvent<any>) => {
        console.log(dc)
      }

      const offer = await peerConnection.createOffer()

      await peerConnection.setLocalDescription(offer)

      const response: IngestResponse = await fetch('http://localhost:8080/ingress/whip/session/test', {
        method: 'POST',
        body: JSON.stringify({
          offer: {
            sdp: offer.sdp,
            type: offer.type,
          }
        })
      }).then(r => r.json())

      const desc = new RTCSessionDescription({
        type: 'answer',
        sdp: response.answer.sdp,
      })

      await peerConnection.setRemoteDescription(desc)

    } catch (err) {
      console.log(err)
    }
  }, [peerConnection])

  const getInfo = useCallback(() => {
    console.log("senders", peerConnection.getSenders())
    console.log("receivers", peerConnection.getReceivers())
    console.log("transceivers", peerConnection.getTransceivers())
  }, [peerConnection])

  const stop = useCallback(() => {
    peerConnection.close()
    setPeerConnection(undefined as never)
  }, [peerConnection]);

  return {
    getInfo,
    ingest,
    stop
  }
}

function Broadcast() {
  const { peerConnection } = useContext(PeerConnectionContext)

  function Broadcast() {
    navigator.mediaDevices.getUserMedia({ video: true, audio: true })
      .then(stream => {
        const pc = peerConnection
        pc.ontrack = function(event) {
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

        document.getElementById('localVideo').srcObject = stream

        stream.getTracks().forEach(track => pc.addTrack(track, stream))

      }).catch(window.alert)

  }


  return (
    <>
      <button onClick={Broadcast}>Broadcast</button>
      <h3> Local Video </h3>
      <video id="localVideo" width="160" height="120" autoplay muted></video> <br />

      <h3> Remote Video </h3>
      <div id="remoteVideos"></div> <br />

      <h3> Logs </h3>
      <div id="logs"></div>

    </>
  )
}

function Main() {
  const { ingest, stop, getInfo } = useWebrtcIngest()

  return (
    <>
      <Broadcast />
      <button onClick={ingest}>Ingest</button>
      <button onClick={stop}>Stop</button>
      <button onClick={getInfo}>Info</button>
    </>
  )
}


function App() {
  const [count, setCount] = useState(0)

  useEffect(() => console.log(count), [count])

  return (
    <>
      <Main />
      <div>
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
