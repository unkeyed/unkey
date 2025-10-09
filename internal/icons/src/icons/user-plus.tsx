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
import { type IconProps, sizeMap } from "../props";

export const UserPlus: React.FC<IconProps> = ({
  iconSize = "xl-thin",
  ...props
}) => {
  const { iconSize: pixelSize, strokeWidth } = sizeMap[iconSize];

  return (
    <svg
      height={pixelSize}
      width={pixelSize}
      {...props}
      viewBox="0 0 24 24"
      xmlns="http://www.w3.org/2000/svg"
    >
      <g
        fill="none"
        strokeLinecap="round"
        strokeLinejoin="round"
        transform="translate(0.25 0.25)"
      >
        <circle
          cx="10"
          cy="6"
          fill="none"
          r="4"
          stroke="currentcolor"
          strokeWidth={strokeWidth}
        />
        <path
          d="m10,13c-4.418,0-8,3.582-8,8,5.333,1.333,10.667,1.333,16,0,0-4.418-3.582-8-8-8Z"
          fill="none"
          stroke="currentcolor"
          strokeWidth={strokeWidth}
        />
        <line
          fill="none"
          stroke="currentColor"
          strokeWidth={strokeWidth}
          x1="20"
          x2="20"
          y1="14"
          y2="8"
        />
        <line
          fill="none"
          stroke="currentColor"
          strokeWidth={strokeWidth}
          x1="17"
          x2="23"
          y1="11"
          y2="11"
        />
      </g>
    </svg>
  );
};
