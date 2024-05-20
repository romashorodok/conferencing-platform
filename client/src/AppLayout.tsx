import { Dispatch, FormEvent, SetStateAction, createRef, useCallback, useContext, useState } from "react"
import { Room, RoomNotifierContext, createRoom } from "./room/RoomProvider"
import * as Dialog from '@radix-ui/react-dialog'
import { PlusIcon } from "@radix-ui/react-icons";
import { NavLink } from "react-router-dom";
import { useSize } from "./utils/resize";
import { useAuth, useAuthorizedFetch } from "./rtc/AuthProvider";
import { IDENTITY_SERVER } from "./variables";
import * as yup from 'yup';
import { useForm } from "./hooks/useForm";
import * as Form from "@radix-ui/react-form";
import { FormField } from "./base/form-field";

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

export function GalleryIcon({ className = "p-[0] w-6 w-6" }: { className?: string }) {
  return (
    <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" strokeWidth={1.5} stroke="currentColor" className={className}>
      <path strokeLinecap="round" strokeLinejoin="round" d="m2.25 15.75 5.159-5.159a2.25 2.25 0 0 1 3.182 0l5.159 5.159m-1.5-1.5 1.409-1.409a2.25 2.25 0 0 1 3.182 0l2.909 2.909m-18 3.75h16.5a1.5 1.5 0 0 0 1.5-1.5V6a1.5 1.5 0 0 0-1.5-1.5H3.75A1.5 1.5 0 0 0 2.25 6v12a1.5 1.5 0 0 0 1.5 1.5Zm10.5-11.25h.008v.008h-.008V8.25Zm.375 0a.375.375 0 1 1-.75 0 .375.375 0 0 1 .75 0Z" />
    </svg>

  )
}

export function UserIcon({ className = "p-[0] w-6 w-6" }: { className?: string }) {
  return (
    <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" strokeWidth={1.5} stroke="currentColor" className={className}>
      <path strokeLinecap="round" strokeLinejoin="round" d="M15.75 6a3.75 3.75 0 1 1-7.5 0 3.75 3.75 0 0 1 7.5 0ZM4.501 20.118a7.5 7.5 0 0 1 14.998 0A17.933 17.933 0 0 1 12 21.75c-2.676 0-5.216-.584-7.499-1.632Z" />
    </svg>
  )
}

export function StopIcon({ className = "p-[0] w-6 w-6" }: { className?: string }) {
  return (
    <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" className={className}>
      <path stroke-linecap="round" stroke-linejoin="round" d="M18.364 18.364A9 9 0 0 0 5.636 5.636m12.728 12.728A9 9 0 0 1 5.636 5.636m12.728 12.728L5.636 5.636" />
    </svg>

  )
}

export function DialogWindow(props: {
  open: boolean,
  setOpen: Dispatch<SetStateAction<boolean>>,
  button: JSX.Element,
  title: JSX.Element,
  description: JSX.Element,
  content: JSX.Element
}) {

  return (
    <Dialog.Root open={props.open}>
      <Dialog.Trigger asChild>
        {props.button}
      </Dialog.Trigger>
      <Dialog.Portal>
        <Dialog.Overlay className="DialogOverlay" />
        <Dialog.Content className="DialogContent max-w-[450px] max-h-[85vh] z-[1000]">
          <Dialog.Title className="DialogTitle">
            {props.title}
          </Dialog.Title>
          <Dialog.Description className="DialogDescription">
            {props.description}
          </Dialog.Description>
          {props.content}
          <Dialog.Close asChild>
            <button className="IconButton cursor-pointer" aria-label="Close" onClick={() => props.setOpen(false)}>
              <CloseIcon />
            </button>
          </Dialog.Close>
        </Dialog.Content>
      </Dialog.Portal>
    </Dialog.Root>
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

  const button =
    <button className="filter-white cursor-pointer px-4" onClick={() => setOpen(true)}>
      <PlusIcon className="filter-white p-[0]" width={18} height={18} />
    </button>

  const title = <p>Create conference room</p>

  const description = <p>Create conferencing room to collaborate</p>

  const content =
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


  return <DialogWindow button={button} open={open} setOpen={setOpen} title={title} description={description} content={content} />
}

const signInFormSchema = yup.object({
  username: yup.string().required("Username required").min(2, "Username must be at least 2 characters"),
  password: yup.string().required("Password required"),
})

export function SignInForm({ setOpen }: { setOpen: Dispatch<SetStateAction<boolean>> }) {
  const { signIn } = useAuth()
  const { fetch } = useAuthorizedFetch()
  const { state, onChange, messages, validate, setMessages } = useForm({
    username: "",
    password: "",
  }, signInFormSchema)

  async function onSubmit(e: FormEvent) {
    e.preventDefault()

    if (!await validate()) {
      console.log("Invalid state", state)
      return
    }

    signIn({ username: state.username, password: state.password })
      .then(() => setOpen(false))
      .catch(async (r: Response) => {
        setMessages({
          password: [(await r.json()).message]
        })
      })

  }

  const wallEcho = useCallback(async () => {
    if (!fetch) return

    const resp = await fetch(`${IDENTITY_SERVER}/wall-echo`, {
      method: 'POST',
    })

    console.log("wall echo resp", resp)
  }, [fetch])

  return (
    <Form.Root autoComplete="off" onSubmit={onSubmit} onChange={onChange}>
      <FormField title={"Username"} name={"username"} type={"text"} messages={messages} />
      <FormField title={"Password"} name={"password"} type={"password"} messages={messages} />
      <Form.Submit asChild>
        <button type="submit"
          className="Button box-border w-full inline-flex h-[35px] items-center justify-center rounded-[4px] px-[15px] font-medium leading-none cursor-pointer focus:outline-none mt-[10px]">
          Sign In
        </button>
      </Form.Submit>

      <button type="button" onClick={wallEcho}>Identity wall echo</button>
    </Form.Root>
  )
}

function SignInDialog({ openDialog = false }) {
  const [open, setOpen] = useState<boolean>(openDialog)

  const button = <button onClick={() => setOpen(true)}>SignUp</button>


  return <DialogWindow open={open} setOpen={setOpen} button={button} title={<p>Sign In</p>} description={<></>} content={<SignInForm setOpen={setOpen} />} />
}

function SignOutButton() {
  const { signOut } = useAuth()

  return (
    <button className={`Button max-w-[70px] px-1 w-full h-[30px] violet clipped-text block`} type="button" onClick={signOut}>Sign Out</button>
  )
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
  const { authenticated } = useAuth()

  return (
    <div ref={rootDivRef} className={`flex flex-col min-h-screen max-h-screen`}>
      <section className={`flex-[20] flex overflow-hidden`}>
        {showNawbar(width) &&
          <div>
            {(() => {
              if (!authenticated) {
                return (
                  <>
                    <SignInDialog />
                  </>
                )
              }
              return (
                <div className={`flex justify-center py-2`}>
                  <SignOutButton />
                </div>
              )
            })()}
            <RoomNav />
          </div>
        }
        <main className={`flex flex-col flex-[1] max-h-full overflow-scroll relative`}>
          {children}
        </main>
      </section>
    </div>
  )
}
