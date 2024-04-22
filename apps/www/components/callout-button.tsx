import Link from "next/link";

export function CalloutButton({ className }: { className?: string }) {
  return (
    <div
      className={`hero-hiring-gradient w-[155ppx] text-white text-sm flex space-x-2 py-1.5 px-2 items-center ${className}`}
    >
      <svg
        xmlns="http://www.w3.org/2000/svg"
        width="16"
        height="16"
        viewBox="0 0 16 16"
        fill="none"
      >
        <g clipPath="url(#clip0_840_5284)">
          <path
            d="M13 0.75C13 1.89705 11.8971 3 10.75 3C11.8971 3 13 4.10295 13 5.25C13 4.10295 14.1029 3 15.25 3C14.1029 3 13 1.89705 13 0.75Z"
            stroke="white"
            strokeLinecap="round"
            strokeLinejoin="round"
          />
          <path
            d="M13 10.75C13 11.8971 11.8971 13 10.75 13C11.8971 13 13 14.1029 13 15.25C13 14.1029 14.1029 13 15.25 13C14.1029 13 13 11.8971 13 10.75Z"
            stroke="white"
            strokeLinecap="round"
            strokeLinejoin="round"
          />
          <path
            d="M5 3.75C5 5.91666 2.91666 8 0.75 8C2.91666 8 5 10.0833 5 12.25C5 10.0833 7.0833 8 9.25 8C7.0833 8 5 5.91666 5 3.75Z"
            stroke="white"
            strokeLinecap="round"
            strokeLinejoin="round"
          />
        </g>
        <defs>
          <clipPath id="clip0_840_5284">
            <rect width="16" height="16" fill="white" />
          </clipPath>
        </defs>
      </svg>
      <Link href="/careers">We are hiring!</Link>
      <svg
        xmlns="http://www.w3.org/2000/svg"
        width="16"
        height="16"
        viewBox="0 0 16 16"
        fill="none"
      >
        <path d="M12 8.5L8 4.5M12 8.5L8 12.5M12 8.5H4" stroke="white" />
      </svg>
    </div>
  );
}
