import { type IconProps, sizeMap } from "../props";

export function Unkey({ iconSize = "xl-thin", ...props }: IconProps) {
  const { iconSize: pixelSize } = sizeMap[iconSize];

  return (
    <svg
      xmlns="http://www.w3.org/2000/svg"
      width={pixelSize}
      height={pixelSize}
      viewBox="100 100 310 310"
      {...props}
    >
      <g fill="currentColor">
        <path d="M170.8 115V340.6H341.2L284.4 397H170.8C139.418 397 114 371.761 114 340.6V115H170.8Z" />
        <path d="M398 284.2L341.2 340.6V115H398V284.2Z" />
      </g>
    </svg>
  );
}
