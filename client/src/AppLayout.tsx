import { FormEvent, useContext, useState } from "react"
import { Room, RoomContext, createRoom } from "./room/RoomProvider"
import * as Dialog from '@radix-ui/react-dialog'
import { PlusIcon } from "@radix-ui/react-icons";
import { SignalContext } from "./rtc/SignalProvider";
import { SubscriberContext } from "./rtc/SubscriberProvider";
import { useRoom } from "./App";

function CloseIcon() {
  return (
    <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" strokeWidth={1.5} stroke="currentColor" className="w-6 h-6 white-filter">
      <path strokeLinecap="round" strokeLinejoin="round" d="M6 18 18 6M6 6l12 12" />
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
        <Dialog.Content className="DialogContent z-[1000]">
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
  // const { setSignalServer } = useContext(SignalContext)
  // const { disconnect } = useContext(SubscriberContext)
  const { join } = useRoom()

  function changeRoom(roomID: string) {
    join(roomID)
    // disconnect()
    // setSignalServer(room.roomId)
  }

  return (
    <div className={`p-2`}
      onClick={() => changeRoom(room.roomId)}>
      {JSON.stringify(room)}
    </div>
  )
}

function RoomNav() {
  const { rooms } = useContext(RoomContext)
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

export default function({ children }: React.PropsWithChildren<{}>) {
  return (
    <div className={`flex flex-col min-h-screen max-h-screen`}>
      <header>Some link or login</header>
      <section className={`flex-[1] flex overflow-hidden`}>
        <RoomNav />
        <main className={`flex flex-col flex-[1] min-h-inherit overflow-scroll`}>
          {children}
        </main>
      </section>
    </div>
  )
}
