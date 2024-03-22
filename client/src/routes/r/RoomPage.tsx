import { useContext, useEffect, useState } from "react"
import { useParams } from "react-router-dom"
import { CameraComponent, Filter, RoomStream, useRoom } from "../../App"
import { MediaStreamContext } from "../../rtc/MediaStreamProvider"
import * as ContextMenu from '@radix-ui/react-context-menu';

type videoFiltersMenuProps = {
  videoFilterList: Array<Filter>
  setVideoFilter: (filter: Filter) => Promise<void>
};

function VideoFiltersMenu({ videoFilterList, setVideoFilter }: videoFiltersMenuProps) {
  const [selected, setSelected] = useState<string | undefined>()

  useEffect(() => {
  }, [videoFilterList])

  const onChangeVideoFilter = (filterIdx: string) => {
    const idx = parseInt(filterIdx)
    setVideoFilter(videoFilterList[idx])
  }

  return (
    <ContextMenu.Root>
      <ContextMenu.Trigger>Open video filters</ContextMenu.Trigger>
      <ContextMenu.Portal>
        <ContextMenu.Content className={`z-20`}>
          <ContextMenu.Item className="ContextMenuItem">
            Close
          </ContextMenu.Item>
          <ContextMenu.Separator />

          <ContextMenu.Label className="ContextMenuLabel">Select video filter</ContextMenu.Label>
          <ContextMenu.RadioGroup value={selected} onValueChange={onChangeVideoFilter}>
            {videoFilterList.map((filter, idx) => (
              <ContextMenu.RadioItem className="ContextMenuRadioItem" value={idx.toString()} key={idx}>
                <ContextMenu.ItemIndicator className="px-1">
                  Active
                </ContextMenu.ItemIndicator>
                {filter.name}
              </ContextMenu.RadioItem>
            ))}
          </ContextMenu.RadioGroup>
        </ContextMenu.Content>
      </ContextMenu.Portal>
    </ContextMenu.Root>
  )
}

type PageData = {
  roomID: string
}

function RoomPage() {
  const { roomID } = useParams<PageData>()
  const { join, roomMediaStreamList, videoFilterList, setVideoFilter } = useRoom()
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

      <VideoFiltersMenu videoFilterList={videoFilterList} setVideoFilter={setVideoFilter} />

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
