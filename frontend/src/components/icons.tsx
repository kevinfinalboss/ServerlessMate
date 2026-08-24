import type { SVGProps } from 'react'

type IconProps = SVGProps<SVGSVGElement>

function base(children: React.ReactNode, props: IconProps) {
  return (
    <svg
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth={1.75}
      strokeLinecap="round"
      strokeLinejoin="round"
      {...props}
    >
      {children}
    </svg>
  )
}

export function IconPlay(props: IconProps) {
  return base(<polygon points="7,5 19,12 7,19" />, props)
}

export function IconTrophy(props: IconProps) {
  return base(
    <>
      <path d="M7 4h10v5a5 5 0 0 1-10 0V4Z" />
      <path d="M7 5H4a1 1 0 0 0-1 1v1a4 4 0 0 0 4 4" />
      <path d="M17 5h3a1 1 0 0 1 1 1v1a4 4 0 0 1-4 4" />
      <path d="M12 14v3" />
      <path d="M9 20h6" />
      <path d="M10 17h4l.6 3H9.4l.6-3Z" />
    </>,
    props,
  )
}

export function IconHistory(props: IconProps) {
  return base(
    <>
      <path d="M4 12a8 8 0 1 0 2.6-5.9" />
      <polyline points="4,4 4,8.5 8.5,8.5" />
      <polyline points="12,8 12,12.5 15,14.5" />
    </>,
    props,
  )
}

export function IconUsers(props: IconProps) {
  return base(
    <>
      <circle cx="9" cy="8" r="3.2" />
      <path d="M3.5 19a5.5 5.5 0 0 1 11 0" />
      <path d="M16 8.2a3.2 3.2 0 1 1 0 6.4" />
      <path d="M15.5 13.5c2.6.4 4.5 2.4 4.9 5.5" />
    </>,
    props,
  )
}

export function IconUser(props: IconProps) {
  return base(
    <>
      <circle cx="12" cy="8" r="3.5" />
      <path d="M5 19.5a7 7 0 0 1 14 0" />
    </>,
    props,
  )
}

export function IconLogOut(props: IconProps) {
  return base(
    <>
      <path d="M9 19H5a1 1 0 0 1-1-1V6a1 1 0 0 1 1-1h4" />
      <polyline points="15,16 20,12 15,8" />
      <line x1="20" y1="12" x2="9" y2="12" />
    </>,
    props,
  )
}

export function IconUserPlus(props: IconProps) {
  return base(
    <>
      <circle cx="9" cy="8" r="3.2" />
      <path d="M2.5 19a6.5 6.5 0 0 1 13 0" />
      <line x1="18" y1="6" x2="18" y2="12" />
      <line x1="15" y1="9" x2="21" y2="9" />
    </>,
    props,
  )
}

export function IconCheck(props: IconProps) {
  return base(<polyline points="4,12.5 9,17.5 20,6" />, props)
}

export function IconX(props: IconProps) {
  return base(
    <>
      <line x1="6" y1="6" x2="18" y2="18" />
      <line x1="18" y1="6" x2="6" y2="18" />
    </>,
    props,
  )
}

export function IconClock(props: IconProps) {
  return base(
    <>
      <circle cx="12" cy="12" r="8.5" />
      <polyline points="12,7.5 12,12 15.5,14" />
    </>,
    props,
  )
}

export function IconSwords(props: IconProps) {
  return base(
    <>
      <path d="M5 5l7 7" />
      <path d="M5 5l3-1 1 3-3 1-1-3Z" />
      <path d="M19 5l-7 7" />
      <path d="M19 5l-3-1-1 3 3 1 1-3Z" />
      <path d="M9 15l-4.5 4.5" />
      <path d="M15 15l4.5 4.5" />
      <path d="M12 12l1.5 1.5-1.5 1.5-1.5-1.5Z" />
    </>,
    props,
  )
}

export function IconChevronRight(props: IconProps) {
  return base(<polyline points="9,5 16,12 9,19" />, props)
}
