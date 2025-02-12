/**
 * Copyright © Nucleo
 * Version 1.3, January 3, 2024
 * Nucleo Icons
 * https://nucleoapp.com/
 * - Redistribution of icons is prohibited.
 * - Icons are restricted for use only within the product they are bundled with.
 *
 * For more details:
 * https://nucleoapp.com/license
 */
import type React from "react";

type Size = "sm" | "md" | "lg" | "xl";
type Weight = "thin" | "regular" | "bold";

type IconSize = `${Size}-${Weight}`;

export const sizeMap = {
  "sm-thin": { size: 12, strokeWidth: 1 },
  "sm-regular": { size: 12, strokeWidth: 2 },
  "sm-bold": { size: 12, strokeWidth: 3 },
  "md-thin": { size: 14, strokeWidth: 1 },
  "md-regular": { size: 14, strokeWidth: 2 },
  "md-bold": { size: 14, strokeWidth: 3 },
  "lg-thin": { size: 16, strokeWidth: 1 },
  "lg-regular": { size: 16, strokeWidth: 2 },
  "lg-bold": { size: 16, strokeWidth: 3 },
  "xl-thin": { size: 18, strokeWidth: 1 },
  "xl-regular": { size: 18, strokeWidth: 2 },
  "xl-bold": { size: 18, strokeWidth: 3 },
} as const;

export type IconProps = {
  className?: string;
  title?: string;
  size?: IconSize;
  filled?: boolean;
};
export const ClockRotateClockwise: React.FC<IconProps> = ({ size, filled, ...props }) => {
  const { size: pixelSize, strokeWidth } = sizeMap[size || "md-regular"];

  return (
    <svg
      {...props}
      height={pixelSize}
      width={pixelSize}
      viewBox="0 0 18 18"
      xmlns="http://www.w3.org/2000/svg"
    >
      <g fill="currentColor">
        <polyline
          fill="none"
          points="10 7 10 10 12 12"
          stroke="currentColor"
          strokeLinecap="round"
          strokeLinejoin="round"
          strokeWidth={strokeWidth}
        />
        <polygon
          fill="currentColor"
          points="4.367 16.956 3.771 13.202 7.516 13.855 4.367 16.956"
          stroke="currentColor"
          strokeLinecap="round"
          strokeLinejoin="round"
          strokeWidth={strokeWidth}
        />
        <path
          d="m5,14.899c1.271,1.297,3.041,2.101,5,2.101,3.866,0,7-3.134,7-7s-3.134-7-7-7c-2.792,0-5.203,1.635-6.326,4"
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
