import type { IconProps } from "../props";

export function IconTagOutline12(props: IconProps) {
  return (
    <svg xmlns="http://www.w3.org/2000/svg" width={12} height={12} viewBox="0 0 12 12" {...props}>
      <circle cx="4" cy="4" r="1" fill="currentColor" strokeWidth="0" />
      <path
        d="m1.25,2.25v3.086c0,.265.105.52.293.707l4.105,4.105c.781.781,2.047.781,2.828,0l1.672-1.672c.781-.781.781-2.047,0-2.828L6.043,1.543c-.188-.188-.442-.293-.707-.293h-3.086c-.552,0-1,.448-1,1Z"
        fill="none"
        stroke="currentColor"
        strokeLinecap="round"
        strokeLinejoin="round"
        strokeWidth={1.5}
      />
    </svg>
  );
}
