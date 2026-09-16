import type { IconProps } from "../props";

export function IconWindowLayoutOutline18(props: IconProps) {
  return (
    <svg xmlns="http://www.w3.org/2000/svg" width={18} height={18} viewBox="0 0 18 18" {...props}>
      <line
        x1="6.25"
        y1="7.75"
        x2="6.25"
        y2="15.25"
        fill="none"
        stroke="currentColor"
        strokeLinecap="round"
        strokeLinejoin="round"
        strokeWidth={1.5}
      />
      <line
        x1="1.75"
        y1="7.75"
        x2="16.25"
        y2="7.75"
        fill="none"
        stroke="currentColor"
        strokeLinecap="round"
        strokeLinejoin="round"
        strokeWidth={1.5}
      />
      <rect
        x="1.75"
        y="2.75"
        width="14.5"
        height="12.5"
        rx="2"
        ry="2"
        transform="translate(18 18) rotate(180)"
        fill="none"
        stroke="currentColor"
        strokeLinecap="round"
        strokeLinejoin="round"
        strokeWidth={1.5}
      />
      <circle cx="4.25" cy="5.25" r=".75" fill="currentColor" />
      <circle cx="6.75" cy="5.25" r=".75" fill="currentColor" />
    </svg>
  );
}
