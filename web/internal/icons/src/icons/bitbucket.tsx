import { type IconProps, sizeMap } from "../props";

export function Bitbucket({ iconSize = "xl-thin", ...props }: IconProps) {
  const { iconSize: pixelSize } = sizeMap[iconSize];

  return (
    <svg
      xmlns="http://www.w3.org/2000/svg"
      width={pixelSize}
      height={pixelSize}
      viewBox="0 0 24 24"
      {...props}
    >
      <g fill="currentColor">
        <path d="M0.778 1.213a0.768 0.768 0 0 0-0.768 0.892l3.263 19.81c0.084 0.5 0.515 0.868 1.022 0.87h15.655a0.768 0.768 0 0 0 0.768-0.645l3.27-20.03a0.768 0.768 0 0 0-0.768-0.897zM14.52 15.53H9.522L8.17 8.466h7.561z" />
      </g>
    </svg>
  );
}
