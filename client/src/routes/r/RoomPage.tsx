import { useContext, useEffect } from "react"
import { useParams } from "react-router-dom"
import { CameraComponent, RoomStream, useRoom } from "../../App"
import { MediaStreamContext } from "../../rtc/MediaStreamProvider"

type PageData = {
  roomID: string
}

function RoomPage() {
  const { roomID } = useParams<PageData>()
  const { join, roomMediaStreamList } = useRoom()
  const { onPageMountMediaStreamMutex } = useContext(MediaStreamContext)

  useEffect(() => {
    if (!roomID || !onPageMountMediaStreamMutex)
      return
    console.log(onPageMountMediaStreamMutex);

    (async () => {
      await onPageMountMediaStreamMutex.wait
      join({ roomID })
      console.log(onPageMountMediaStreamMutex)
    })()
  }, [roomID, onPageMountMediaStreamMutex])

  return (
    <div>
      <h1>{roomID}</h1>

      <CameraComponent />

      <div>
        {Object.entries(roomMediaStreamList).map(([id, { stream }]) => (
          <RoomStream key={id} mediaStream={stream} />
        ))}
      </div>
    </div>
  )
}

export default RoomPage
