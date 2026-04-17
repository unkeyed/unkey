import { type IconProps, sizeMap } from "../props";

export function ProgressCircle({ iconSize, ...props }: IconProps) {
  const { iconSize: pixelSize, strokeWidth } = sizeMap[iconSize || "md-regular"];
  return (
    <svg
      height={pixelSize}
      width={pixelSize}
      viewBox="0 0 18 18"
      xmlns="http://www.w3.org/2000/svg"
      {...props}
    >
      <g fill="none" stroke="currentColor" strokeLinecap="round" strokeLinejoin="round">
        <path
          d="M13.163,3.07c-.854-.601-1.843-1.019-2.913-1.205"
          strokeWidth={strokeWidth}
        />
        <path
          d="M16.137,7.75c-.179-1.029-.583-2.023-1.208-2.912"
          strokeWidth={strokeWidth}
        />
        <path
          d="M14.93,13.163c.601-.854,1.019-1.843,1.205-2.913"
          strokeWidth={strokeWidth}
        />
        <path
          d="M10.25,16.137c1.029-.179,2.023-.583,2.912-1.208"
          strokeWidth={strokeWidth}
        />
        <path
          d="M4.837,14.93c.854,.601,1.843,1.019,2.913,1.205"
          strokeWidth={strokeWidth}
        />
        <path
          d="M1.863,10.25c.179,1.029,.583,2.023,1.208,2.912"
          strokeWidth={strokeWidth}
        />
        <path
          d="M3.07,4.837c-.601,.854-1.019,1.843-1.205,2.913"
          strokeWidth={strokeWidth}
        />
        <path
          d="M7.75,1.863c-1.029,.179-2.023,.583-2.912,1.208"
          strokeWidth={strokeWidth}
        />
        <circle cx="9" cy="9" r="2.25" strokeWidth={strokeWidth} />
      </g>
    </svg>
  );
}
