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
import React from "react";
import { type IconProps, sizeMap } from "../props";

export const AdjustContrast3: React.FC<IconProps> = ({
  iconsize = "xl-thin",
  ...props
}) => {
  const { iconsize: pixelSize, strokeWidth } = sizeMap[iconsize];
  return (
    <svg
      height={pixelSize}
      width={pixelSize}
      viewBox="0 0 24 24"
      xmlns="http://www.w3.org/2000/svg"
      {...props}
    >
      <g
        fill="currentColor"
        strokeLinecap="round"
        strokeLinejoin="round"
        transform="translate(0.25 0.25)"
      >
        <line
          fill="none"
          stroke="currentColor"
          strokeWidth={strokeWidth}
          x1="19.071"
          x2="4.929"
          y1="4.929"
          y2="19.071"
        />
        <circle
          cx="12"
          cy="12"
          fill="none"
          r="10"
          stroke="currentColor"
          strokeWidth={strokeWidth}
        />
      </g>
    </svg>
  );
};
