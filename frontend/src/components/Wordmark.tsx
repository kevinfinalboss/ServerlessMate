import { Link } from 'react-router-dom'

export function Wordmark({ className = '' }: { className?: string }) {
  return (
    <Link to="/" className={`font-display text-xl font-medium tracking-tight ${className}`}>
      <span className="text-paper">Serverless</span>
      <span className="text-lime">Mate</span>
    </Link>
  )
}
