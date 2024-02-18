import { Dispatch, PropsWithChildren, Reducer, createContext, useReducer } from "react";

type StreamContext = {
  stream: MediaStream,
  originList: Array<RTCTrackEvent>
}

type MediaStreamContextReducer = { [streamID: string]: StreamContext }

type DeleteStreamEvent = undefined

type RoomMediaTrackAction = {
  action: 'clear' | 'mutate'
  payload: TrackEvent
}
type TrackEvent = { [streamID: string]: RTCTrackEvent | DeleteStreamEvent }

type RoomMediaStreamListContextType = {
  roomMediaStreamList: MediaStreamContextReducer
  setRoomMediaStream: Dispatch<RoomMediaTrackAction>
}

export const RoomMediaStreamListContext = createContext<RoomMediaStreamListContextType>(null!)

function RoomMediaStreamListProvider({ children }: PropsWithChildren<{}>) {
  const [roomMediaStreamList, setRoomMediaStream] = useReducer<Reducer<MediaStreamContextReducer, RoomMediaTrackAction>>(
    (state, event) => {
      switch (event.action) {
        case 'clear': {
          return {}
        }
        case 'mutate': {
          Object.entries(event.payload).forEach(([streamID, trackEvent]) => {
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
        }
        default:
          throw Error("unknown room media track action")
      }
    },
    {}
  )

  return (
    <RoomMediaStreamListContext.Provider value={{ roomMediaStreamList, setRoomMediaStream }}>
      {children}
    </RoomMediaStreamListContext.Provider>
  )
}

export default RoomMediaStreamListProvider;
