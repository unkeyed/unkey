import type { IconProps } from "../props";

export function IconDotsOutline18(props: IconProps) {
  return (
    <svg xmlns="http://www.w3.org/2000/svg" width={18} height={18} viewBox="0 0 18 18" {...props}>
      <circle
        cx="9"
        cy="9"
        r=".5"
        fill="currentColor"
        stroke="currentColor"
        strokeLinecap="round"
        strokeLinejoin="round"
        strokeWidth={1.5}
      />
      <circle
        cx="3.25"
        cy="9"
        r=".5"
        stroke="currentColor"
        strokeLinecap="round"
        strokeLinejoin="round"
        strokeWidth={1.5}
        fill="currentColor"
      />
      <circle
        cx="14.75"
        cy="9"
        r=".5"
        stroke="currentColor"
        strokeLinecap="round"
        strokeLinejoin="round"
        strokeWidth={1.5}
        fill="currentColor"
      />
    </svg>
  );
}
