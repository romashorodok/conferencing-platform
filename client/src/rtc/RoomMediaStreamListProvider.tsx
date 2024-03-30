import { Dispatch, PropsWithChildren, Reducer, createContext, useReducer } from "react";
import { Mutex } from "../App";

type StreamContext = {
  stream: MediaStream,
}

export type MediaStreamContextReducer = { [streamID: string]: StreamContext }

type DeleteStreamEvent = undefined

type RoomMediaTrackAction = {
  action: 'clear' | 'mutate' | 'remove'
  payload: TrackEvent
}
type TrackEvent = { [streamID: string]: RTCTrackEvent | MediaStreamTrackEvent | DeleteStreamEvent }

type RoomMediaStreamListContextType = {
  roomMediaStreamList: Promise<MediaStreamContextReducer>
  setRoomMediaStream: Dispatch<RoomMediaTrackAction>
}

export const RoomMediaStreamListContext = createContext<RoomMediaStreamListContextType>(null!)


function RoomMediaStreamListProvider({ children }: PropsWithChildren<{}>) {
  const mutex = new Mutex()

  const [roomMediaStreamList, setRoomMediaStream] = useReducer<Reducer<Promise<MediaStreamContextReducer>, RoomMediaTrackAction>>(
    async (reducer, event) => {
      await mutex.wait
      const unlock = mutex.lock()
      const state = await reducer

      switch (event.action) {
        case 'clear': {
          (await unlock)()
          return {}
        }

        case 'remove': {
          Object.entries(event.payload).forEach(([streamID, trackEvent]) => {
            if (trackEvent instanceof RTCTrackEvent) return
            if (!trackEvent?.track) return
            if (!state[streamID]) return

            const context = state[streamID]
            context.stream.removeTrack(trackEvent.track)
            if (context.stream.getTracks().length == 0) {
              delete state[streamID]
            }
          });

          (await unlock)()
          return { ...state }
        }

        case 'mutate': {
          Object.entries(event.payload).forEach(([streamID, trackEvent]) => {
            if (trackEvent instanceof MediaStreamTrackEvent) return

            if (!trackEvent) {
              delete state[streamID]
              return
            }

            let context: StreamContext
            if (state[streamID]) {
              context = state[streamID]
            } else {
              context = { stream: trackEvent.streams[0] }
            }

            context.stream = trackEvent.streams[0]

            Object.assign(state, { [streamID]: context })
          });

          (await unlock)()
          return { ...state }
        }
        default:
          throw Error("unknown room media track action")
      }
    },
    Promise.resolve({})
  )

  return (
    <RoomMediaStreamListContext.Provider value={{ roomMediaStreamList, setRoomMediaStream }}>
      {children}
    </RoomMediaStreamListContext.Provider>
  )
}

export default RoomMediaStreamListProvider;
