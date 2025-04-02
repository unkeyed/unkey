import type React from "react";
import { type IconProps, sizeMap } from "../props";

export const Clock: React.FC<IconProps> = ({ size, filled, ...props }) => {
  const { size: pixelSize, strokeWidth } = sizeMap[size || "md-regular"];
  return (
    <svg
      height={pixelSize}
      width={pixelSize}
      {...props}
      viewBox="0 0 18 18"
      xmlns="http://www.w3.org/2000/svg"
      {...props}
    >
      <g fill="currentColor">
        <circle
          cx="9"
          cy="9"
          fill={filled ? "currentColor" : "none"}
          r="7.25"
          stroke="currentColor"
          strokeLinecap="round"
          strokeLinejoin="round"
          strokeWidth={strokeWidth}
        />
        <polyline
          fill={filled ? "currentColor" : "none"}
          points="9 4.75 9 9 12.25 11.25"
          stroke="currentColor"
          strokeLinecap="round"
          strokeLinejoin="round"
          strokeWidth={strokeWidth}
        />
      </g>
    </svg>
  );
};
