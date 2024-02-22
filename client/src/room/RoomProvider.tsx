import { PropsWithChildren, createContext, useEffect, useState } from "react";
import { debounce } from "../utils/debounce";
import { EventEmitter } from 'events';
import { MEDIA_SERVER, MEDIA_SERVER_WS } from "../variables";

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

export type Room = {
  participants: Array<Participant>,
  roomId: string,
}

type RoomContextType = {
  rooms: Array<Room>
};

export const RoomNotifierContext = createContext<RoomContextType>(undefined!)

const ROOMS_ENDPOINT = `${MEDIA_SERVER}/rooms`

const ROOM_NOTIFIER_ENDPOINT = `${MEDIA_SERVER_WS}/rooms-notifier`

export async function createRoom(body: { roomId: string, maxParticipants: number }) {
  return fetch(`${MEDIA_SERVER}/rooms`, {
    method: 'POST',
    body: JSON.stringify(body),
  })
}

function RoomNotifierContextProvider({ children }: PropsWithChildren<{}>) {
  const [rooms, setRooms] = useState<Array<Room>>([])
  const [notifier,] = useState<RoomsNotifier>(new RoomsNotifier(ROOM_NOTIFIER_ENDPOINT))

  function updateRooms() {
    fetch(ROOMS_ENDPOINT)
      .then(r => r.json())
      .then(({ rooms = [] }) => setRooms(rooms))
  }

  function deferUpdateRooms(): (...args: any) => void {
    return debounce(updateRooms, 200);
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
    <RoomNotifierContext.Provider value={{ rooms }}>
      {children}
    </RoomNotifierContext.Provider>
  )
}

export default RoomNotifierContextProvider;
