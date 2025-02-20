import type React from "react";
import { type IconProps, sizeMap } from "../props";

export const ArrowTriangleLineDown: React.FC<IconProps> = ({ size = "xl-thin", ...props }) => {
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
          y1="10.25"
          y2="2.25"
        />
        <path
          d="M9.414,15.547l3.058-4.516c.225-.332-.013-.78-.414-.78H5.942c-.401,0-.639,.448-.414,.78l3.058,4.516c.198,.293,.63,.293,.828,0Z"
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
