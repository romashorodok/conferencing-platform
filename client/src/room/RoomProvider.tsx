import { PropsWithChildren, createContext, useEffect, useState } from "react";
import { debounce } from "../utils/debounce";
import { EventEmitter } from 'events';

enum RoomsNotifierEvent {
  UPDATE_ROOMS = 'update-rooms',
}

class RoomsNotifier extends EventEmitter {
  ws?: WebSocket

  constructor(private readonly uri: string = "ws://localhost:8080/rooms-notifier") {
    super()
  }

  connect() {
    this.ws = new WebSocket(this.uri)
    this.ws.onmessage = (ev) => {
      const { event = null, data = null } = JSON.parse(ev.data)
      if (!event) {
        return
      }
      this.emit(event, data)
    }
  }

  close() {
    this.ws?.close()
  }
}

type Participant = {}

type Room = {
  participants: Array<Participant>,
  roomId: string,
}

type RoomContextType = {
  rooms: Array<Room>
};

export const RoomContext = createContext<RoomContextType>(undefined!)

const MEDIA_SERVER = import.meta.env.VITE_MEDIA_SERVER
const ROOMS_ENDPOINT = `${MEDIA_SERVER}/rooms`

const MEDIA_SERVER_WS = import.meta.env.VITE_MEDIA_SERVER_WS
const ROOM_NOTIFIER_ENDPOINT = `${MEDIA_SERVER_WS}/rooms-notifier`

function RoomContextProvider({ children }: PropsWithChildren<{}>) {
  const [rooms, setRooms] = useState<Array<Room>>([])
  const [notifier,] = useState<RoomsNotifier>(new RoomsNotifier(ROOM_NOTIFIER_ENDPOINT))

  function updateRooms() {
    fetch(ROOMS_ENDPOINT)
      .then(r => r.json())
      .then(({ rooms = [] }) => setRooms(rooms))
  }

  function deferUpdateRooms(): (...args: any) => void {
    return debounce(updateRooms, 800);
  }

  useEffect(() => {
    updateRooms()
  }, [])

  useEffect(() => {
    if (notifier) {
      notifier.connect()
      const update = deferUpdateRooms()
      notifier.on(RoomsNotifierEvent.UPDATE_ROOMS, function() {
        update()
      })
    }
    return () => {
      if (notifier) {
        notifier.removeAllListeners(RoomsNotifierEvent.UPDATE_ROOMS)
        notifier.close()
      }
    }
  }, [notifier])

  return (
    <RoomContext.Provider value={{ rooms }}>
      {children}
    </RoomContext.Provider>
  )
}

export default RoomContextProvider;
