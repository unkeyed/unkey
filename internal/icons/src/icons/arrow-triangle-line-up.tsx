import type React from "react";
import { type IconProps, sizeMap } from "../props";

export const ArrowTriangleLineUp: React.FC<IconProps> = ({ size = "xl-thin", ...props }) => {
  const { size: pixelSize, strokeWidth } = sizeMap[size];
  return (
    <svg
      {...props}
      height={pixelSize}
      width={pixelSize}
      viewBox="0 0 18 18"
      xmlns="http://www.w3.org/2000/svg"
    >
      <g fill="currentColor">
        <line
          fill="none"
          stroke="currentColor"
          strokeLinecap="round"
          strokeLinejoin="round"
          strokeWidth={strokeWidth}
          x1="9"
          x2="9"
          y1="7.75"
          y2="15.75"
        />
        <path
          d="M8.586,2.453l-3.058,4.516c-.225,.332,.013,.78,.414,.78h6.115c.401,0,.639-.448,.414-.78l-3.058-4.516c-.198-.293-.63-.293-.828,0Z"
          fill="none"
          stroke="currentColor"
          strokeLinecap="round"
          strokeLinejoin="round"
          strokeWidth={strokeWidth}
        />
      </g>
    </svg>
  );
};
