import { useEffect } from "react"
import { useParams } from "react-router-dom"

type PageData = {
  roomID: string
}

function RoomPage() {
  const { roomID } = useParams<PageData>()

  useEffect(() => { console.log(roomID) }, [roomID])

  return (
    <div>
      {roomID}
    </div>
  )
}

export default RoomPage
