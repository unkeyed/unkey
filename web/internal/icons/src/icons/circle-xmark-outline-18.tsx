import type { IconProps } from "../props";

export function IconCircleXmarkOutline18(props: IconProps) {
  return (
    <svg xmlns="http://www.w3.org/2000/svg" width={18} height={18} viewBox="0 0 18 18" {...props}>
      <circle
        cx="9"
        cy="9"
        r="7.25"
        fill="none"
        stroke="currentColor"
        strokeLinecap="round"
        strokeLinejoin="round"
        strokeWidth={1.5}
      />
      <line
        x1="6.25"
        y1="6.25"
        x2="11.75"
        y2="11.75"
        fill="none"
        stroke="currentColor"
        strokeLinecap="round"
        strokeLinejoin="round"
        strokeWidth={1.5}
      />
      <line
        x1="11.75"
        y1="6.25"
        x2="6.25"
        y2="11.75"
        fill="none"
        stroke="currentColor"
        strokeLinecap="round"
        strokeLinejoin="round"
        strokeWidth={1.5}
      />
    </svg>
  );
}
