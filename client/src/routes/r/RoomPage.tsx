import { PropsWithChildren, createRef, useContext, useEffect, useMemo, } from "react"
import { useParams } from "react-router-dom"
import { AudioControlLarge, RoomStream, VideoControlLarge, useRoom } from "../../App"
import { MediaStreamContext } from "../../rtc/MediaStreamProvider"
import { useSize } from "../../utils/resize";
import { StopIcon } from "../../AppLayout";

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
    <div ref={gridRef} className={`grid gap-6 min-h-full min-w-full`} style={{
      gridTemplateColumns: `repeat(${columns}, minmax(0, 1fr))`,
      gridTemplateRows: `repeat(${rows}, minmax(0, 1fr))`,
    }} >
      {children}
    </div>
  )
}

export function FaceDetectionButtons() {
  const {  startNormal } = useContext(MediaStreamContext)

  return (
    <>
      <button className="Button cursor-pointer px-2" onClick={() => startNormal()}>
        <StopIcon className="w-[32px] h-[32px]" />
      </button>
    </>
  )
}

type PageData = {
  roomID: string
}

function RoomPage() {
  const { roomID } = useParams<PageData>()
  const { join, roomMediaList, } = useRoom()
  const { onPageMountMediaStreamMutex } = useContext(MediaStreamContext)

  const roomMediaItems = useMemo(() => Object.entries(roomMediaList), [roomMediaList])

  useEffect(() => {
    if (!roomID || !onPageMountMediaStreamMutex)
      return
    console.log(onPageMountMediaStreamMutex);

    (async () => {
      await onPageMountMediaStreamMutex.wait
      // Provide here the media stream is wrong approach. it will be trigger the join
      join({ roomID })
      console.log(onPageMountMediaStreamMutex)
    })()
  }, [roomID, onPageMountMediaStreamMutex])

  return (
    <div className={`relative flex flex-col w-full h-full p-4`}>
      <div className={`flex-1`}>
        <GridLayout itemCount={roomMediaItems.length}>
          {roomMediaItems.map(([id, { stream }]) => (
            id === 'inactive'
              ? null
              : <RoomStream key={id} mediaStream={stream} />
          ))}
        </GridLayout>
      </div>
      <div className={`flex flex-row justify-center gap-4 flex-3 p-4 z-50`}>
        <AudioControlLarge className={`ButtonShadow`} />
        <VideoControlLarge className={`ButtonShadow`} />
      </div>
    </div>
  )
}


export default RoomPage
