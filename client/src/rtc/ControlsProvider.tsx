import { Dispatch, PropsWithChildren, SetStateAction, createContext, useContext, useEffect, useState } from "react";
import { MediaStreamContext, setAudioMute, setVideoMute } from "./MediaStreamProvider";

type ControlsContextProvider = {
  isVideoMuted: boolean,
  isAudioMuted: boolean,
  setVideoMuted: Dispatch<SetStateAction<boolean>>
  setAudioMuted: Dispatch<SetStateAction<boolean>>
}

export const ControlsContext = createContext<ControlsContextProvider>({
  isAudioMuted: true,
  isVideoMuted: true,
  setVideoMuted: () => undefined,
  setAudioMuted: () => undefined,
})

function ControlsProvider({ children }: PropsWithChildren<{}>) {
  const { mediaStream } = useContext(MediaStreamContext)

  const [isVideoMuted, setVideoMuted] = useState<boolean>(!mediaStream?.getVideoTracks()[0]?.enabled || false);
  const [isAudioMuted, setAudioMuted] = useState<boolean>(!mediaStream?.getAudioTracks()[0]?.enabled || false);

  useEffect(() => {
    if (mediaStream)
      setVideoMute(mediaStream, isVideoMuted)
  }, [isVideoMuted, mediaStream])

  useEffect(() => {
    if (mediaStream)
      setAudioMute(mediaStream, isAudioMuted)
  }, [isAudioMuted, mediaStream])

  return (
    <ControlsContext.Provider value={{
      isVideoMuted,
      isAudioMuted,
      setAudioMuted,
      setVideoMuted,
    }}>
      {children}
    </ControlsContext.Provider>
  )
}

export default ControlsProvider;
