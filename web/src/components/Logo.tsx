interface LogoProps {
  className?: string
}

export function Logo({ className = "w-6 h-6" }: LogoProps) {
  return (
    <svg
      width="32"
      height="32"
      viewBox="0 0 32 32"
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
      className={className}
    >
      {/* Abstract Gateway Logo: Simple, minimalist gate design */}
      {/* Left pillar */}
      <rect x="7" y="9" width="2.5" height="15" rx="1.25" fill="currentColor" />
      {/* Right pillar */}
      <rect x="22.5" y="9" width="2.5" height="15" rx="1.25" fill="currentColor" />
      {/* Arch - smooth curved top */}
      <path
        d="M 9.5 9 Q 16 3 22.5 9"
        stroke="currentColor"
        strokeWidth="2.5"
        fill="none"
        strokeLinecap="round"
        strokeLinejoin="round"
      />
      {/* Horizontal connection beam */}
      <line
        x1="9.5"
        y1="13"
        x2="22.5"
        y2="13"
        stroke="currentColor"
        strokeWidth="2"
        strokeLinecap="round"
      />
    </svg>
  )
}
