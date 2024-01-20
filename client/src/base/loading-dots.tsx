import './loading-dots.css'

export default function({ absolute, className = '' }: { absolute: boolean, className?: string }) {
  return (
    <div className={`${absolute ? 'absolute inset-0' : ''} loading ${className}`}>
      <div className="flex m-auto">
        <span className="dot" />
        <span className="dot" />
        <span className="dot" />
      </div>
    </div>
  )
}
