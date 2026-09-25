import type { IconProps } from "../props";

export function IconDotsOutline12(props: IconProps) {
  return (
    <svg xmlns="http://www.w3.org/2000/svg" width={12} height={12} viewBox="0 0 12 12" {...props}>
      <circle cx="6" cy="6" r="1" fill="currentColor" strokeWidth="0" />
      <circle cx="2" cy="6" r="1" strokeWidth="0" fill="currentColor" />
      <circle cx="10" cy="6" r="1" strokeWidth="0" fill="currentColor" />
    </svg>
  );
}
