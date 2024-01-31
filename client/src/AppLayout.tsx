import { useContext } from "react"
import { RoomContext } from "./room/RoomProvider"

function RoomNav() {
  const { rooms } = useContext(RoomContext)
  return (
    <nav className={`w-[140px] overflow-scroll`}>
      {rooms.map((room, idx) => (
        <div key={idx}>{JSON.stringify(room)}</div>
      ))}
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
