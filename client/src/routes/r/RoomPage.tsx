import { PropsWithChildren, createRef, useContext, useEffect, useMemo, useState } from "react"
import { useParams } from "react-router-dom"
import { AudioControl, AudioControlLarge, CameraComponent, Filter, RoomStream, VideoControl, VideoControlLarge, useRoom } from "../../App"
import { MediaStreamContext } from "../../rtc/MediaStreamProvider"
import * as ContextMenu from '@radix-ui/react-context-menu';
import { useSize } from "../../utils/resize";
import * as Dialog from '@radix-ui/react-dialog';
import { PlusIcon } from "@radix-ui/react-icons";
import { CloseIcon, SettingsIcon } from "../../AppLayout";

type videoFiltersMenuProps = {
  videoFilterList: Array<Filter>
  setVideoFilter: (filter: Filter) => Promise<void>
};

function VideoFiltersMenu({ videoFilterList, setVideoFilter }: videoFiltersMenuProps) {
  const [selected, _] = useState<string | undefined>()

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

type GridSize = {
  columns: number,
  rows: number,
  minWidth: number,
  minHeight: number,
  maxItems: number,
  minItems: number,
}

const GridSizes: Array<GridSize> = [
  {
    columns: 1,
    rows: 1,
    maxItems: 1,
    minItems: 1,
    minHeight: 0,
    minWidth: 0
  },
  {
    columns: 1,
    rows: 2,
    maxItems: 2,
    minItems: 2,
    minHeight: 0,
    minWidth: 0
  },
  {
    columns: 2,
    rows: 1,
    maxItems: 2,
    minItems: 1,
    minWidth: 560,
    minHeight: 0,
  },
  {
    columns: 2,
    rows: 2,
    maxItems: 4,
    minItems: 3,
    minWidth: 560,
    minHeight: 0,
  }
]

function selectSize(sizes: typeof GridSizes, itemCount: number, width: number, height: number): GridSize {
  let curr = 0

  let size = sizes.find((currSize, sizeIdx, allSizes) => {
    curr = sizeIdx;
    const isBiggerSizeAvailable = allSizes.findIndex((nextSize, next) => {
      const isNextBiggerThenCurr = next > curr
      const isSameMaxItems = currSize.maxItems == nextSize.maxItems
      return isNextBiggerThenCurr && isSameMaxItems
    }) !== -1

    return currSize.maxItems >= itemCount && !isBiggerSizeAvailable
  })
  if (!size) {
    size = sizes[sizes.length - 1]
    if (size) {
      console.warn(`Not found appropriate size for ${itemCount}. Fallback to ${size.columns}x${size.rows}`)
    } else {
      throw Error("Not found size for layout")
    }
  }

  if (width < size.minWidth || height < size.minHeight) {
    if (curr > 0) {
      const smallerSize = sizes[curr - 1]
      size = selectSize(sizes.slice(0, curr), smallerSize.maxItems, width, height)
    }
  }

  return size
}

function useGridSize({ itemCount, width, height }: {
  itemCount: number,
  width: number,
  height: number,
}): GridSize {

  const size = width > 0 && height > 0
    ? selectSize(GridSizes, itemCount, width, height)
    : GridSizes[0]

  return size
}

function GridLayout({
  children,
  itemCount = 1,
}: PropsWithChildren<{ itemCount: number }>) {
  const gridRef = createRef<HTMLDivElement>()
  const { width, height } = useSize(gridRef)
  const { rows, columns } = useGridSize({ itemCount, width, height })

  return (
    <div ref={gridRef} className={`grid grid-cols-${columns} grid-rows-${rows} gap-2`} >
      {children}
    </div>
  )
}

function SettingsDialog() {
  const [open, setOpen] = useState<boolean>(false)

  return (
    <Dialog.Root open={open}>
      <Dialog.Trigger asChild>
        <button className="Button cursor-pointer px-2" onClick={() => setOpen(true)}>
          <SettingsIcon className="w-[32px] h-[32px]" />
        </button>
      </Dialog.Trigger>
      <Dialog.Portal>
        <Dialog.Overlay className="DialogOverlay" />
        <Dialog.Content className="DialogContent max-w-[530px] max-h-[85vh] z-[1000]">
          <Dialog.Title className="DialogTitle">User settings</Dialog.Title>

          <CameraComponent />

          <Dialog.Close asChild>
            <button className="IconButton cursor-pointer" aria-label="Close" onClick={() => setOpen(false)}>
              <CloseIcon />
            </button>
          </Dialog.Close>
        </Dialog.Content>
      </Dialog.Portal>
    </Dialog.Root>
  )
}

type PageData = {
  roomID: string
}

function RoomPage() {
  const { roomID } = useParams<PageData>()
  const { join, roomMediaList, videoFilterList, setVideoFilter } = useRoom()
  const { onPageMountMediaStreamMutex } = useContext(MediaStreamContext)

  const roomMediaItems = useMemo(() => Object.entries(roomMediaList), [roomMediaList])

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
    <div className={`flex flex-col w-full h-full p-4`}>
      <div className={`flex-1`}>
        <GridLayout itemCount={roomMediaItems.length}>
          {roomMediaItems.map(([id, { stream }]) => (
            <RoomStream key={id} mediaStream={stream} />
          ))}
        </GridLayout>
      </div>
      <div className={`flex flex-row justify-center gap-4 flex-3 p-4`}>
        <AudioControlLarge className={`ButtonShadow`} />
        <VideoControlLarge className={`ButtonShadow`} />
        <SettingsDialog />
        <VideoFiltersMenu videoFilterList={videoFilterList} setVideoFilter={setVideoFilter} />
      </div>
    </div>
  )
}

// <VideoFiltersMenu videoFilterList={videoFilterList} setVideoFilter={setVideoFilter} />
// <h1>{roomID}</h1>

export default RoomPage
