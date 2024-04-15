import { FormEvent, createRef, useContext, useState } from "react"
import { Room, RoomNotifierContext, createRoom } from "./room/RoomProvider"
import * as Dialog from '@radix-ui/react-dialog'
import { PlusIcon } from "@radix-ui/react-icons";
import { NavLink } from "react-router-dom";
import { useSize } from "./utils/resize";

export function CloseIcon() {
  return (
    <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" strokeWidth={1.5} stroke="currentColor" className="w-6 h-6 white-filter">
      <path strokeLinecap="round" strokeLinejoin="round" d="M6 18 18 6M6 6l12 12" />
    </svg>
  )
}

export function SettingsIcon({ className = "p-[0] w-6 w-6" }: { className?: string }) {
  return (
    <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" strokeWidth={1.5} stroke="currentColor" className={className}>
      <path strokeLinecap="round" strokeLinejoin="round" d="M9.594 3.94c.09-.542.56-.94 1.11-.94h2.593c.55 0 1.02.398 1.11.94l.213 1.281c.063.374.313.686.645.87.074.04.147.083.22.127.325.196.72.257 1.075.124l1.217-.456a1.125 1.125 0 0 1 1.37.49l1.296 2.247a1.125 1.125 0 0 1-.26 1.431l-1.003.827c-.293.241-.438.613-.43.992a7.723 7.723 0 0 1 0 .255c-.008.378.137.75.43.991l1.004.827c.424.35.534.955.26 1.43l-1.298 2.247a1.125 1.125 0 0 1-1.369.491l-1.217-.456c-.355-.133-.75-.072-1.076.124a6.47 6.47 0 0 1-.22.128c-.331.183-.581.495-.644.869l-.213 1.281c-.09.543-.56.94-1.11.94h-2.594c-.55 0-1.019-.398-1.11-.94l-.213-1.281c-.062-.374-.312-.686-.644-.87a6.52 6.52 0 0 1-.22-.127c-.325-.196-.72-.257-1.076-.124l-1.217.456a1.125 1.125 0 0 1-1.369-.49l-1.297-2.247a1.125 1.125 0 0 1 .26-1.431l1.004-.827c.292-.24.437-.613.43-.991a6.932 6.932 0 0 1 0-.255c.007-.38-.138-.751-.43-.992l-1.004-.827a1.125 1.125 0 0 1-.26-1.43l1.297-2.247a1.125 1.125 0 0 1 1.37-.491l1.216.456c.356.133.751.072 1.076-.124.072-.044.146-.086.22-.128.332-.183.582-.495.644-.869l.214-1.28Z" />
      <path strokeLinecap="round" strokeLinejoin="round" d="M15 12a3 3 0 1 1-6 0 3 3 0 0 1 6 0Z" />
    </svg>
  )
}
function RoomDialog() {
  const [open, setOpen] = useState<boolean>(false)

  async function onSubmit(evt: FormEvent) {
    evt.preventDefault()
    // @ts-ignore
    const name = evt.target.name.value || undefined

    const resp = await createRoom({ roomId: name, maxParticipants: 4 })

    if (resp.status == 201) {
      setOpen(false)
    }
  }

  return (
    <Dialog.Root open={open}>
      <Dialog.Trigger asChild>
        <button className="filter-white cursor-pointer px-4" onClick={() => setOpen(true)}>
          <PlusIcon className="filter-white p-[0]" width={18} height={18} />
        </button>
      </Dialog.Trigger>
      <Dialog.Portal>
        <Dialog.Overlay className="DialogOverlay" />
        <Dialog.Content className="DialogContent max-w-[450px] max-h-[85vh] z-[1000]">
          <Dialog.Title className="DialogTitle">Create conference room</Dialog.Title>
          <Dialog.Description className="DialogDescription">
            Create conferencing room to collaborate
          </Dialog.Description>
          <form autoComplete="off" onSubmit={onSubmit}>
            <fieldset className="Fieldset">
              <label className="Label" htmlFor="name">
                Name
              </label>
              <input className="Input" name="name" id="name" />
            </fieldset>
            <div style={{ display: 'flex', marginTop: 25, justifyContent: 'flex-end' }}>
              <Dialog.Close asChild>
                <button type="submit" className="cursor-pointer">Create room</button>
              </Dialog.Close>
            </div>
          </form>
          <Dialog.Close asChild>
            <button className="IconButton cursor-pointer" aria-label="Close" onClick={() => setOpen(false)}>
              <CloseIcon />
            </button>
          </Dialog.Close>
        </Dialog.Content>
      </Dialog.Portal>
    </Dialog.Root>
  );
}

function RoomNavItem({ room }: { room: Room }) {
  return (
    <div className={`p-2`}>
      <NavLink to={`/r/${room.roomId}`}>
        <button className={`Button max-w-[100px] px-2 w-full h-[30px] violet clipped-text block`}>
          {room.roomId}
        </button>
      </NavLink>
      <div>
        {JSON.stringify(room.participants)}
      </div>
    </div>
  )
}

function RoomNav() {
  const { rooms } = useContext(RoomNotifierContext)

  return (
    <nav className={`w-[140px] overflow-hidden`}>
      <div className={`flex justify-around mb-4`}>
        <h1 className={`px-4`}>Rooms</h1>
        <RoomDialog />
      </div>
      <div className={`overflow-scroll`}>
        {rooms.map((room, idx) => (
          <RoomNavItem key={idx} room={room} />
        ))}
      </div>
    </nav>
  )
}

function showNawbar(width: number): boolean {
  return width >= 620;
}

export default function({ children }: React.PropsWithChildren<{}>) {
  const rootDivRef = createRef<HTMLDivElement>()
  const { width, } = useSize(rootDivRef)

  return (
    <div ref={rootDivRef} className={`flex flex-col min-h-screen max-h-screen`}>
      <header>
        <NavLink to="/">
          <button>
            Home
          </button>
        </NavLink>
      </header>
      <section className={`flex-[1] flex overflow-hidden`}>
        {showNawbar(width) &&
          <RoomNav />}
        <main className={`flex flex-col flex-[1] min-h-inherit overflow-scroll relative`}>
          {children}
        </main>
      </section>
    </div>
  )
}
