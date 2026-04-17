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

import { type IconProps, sizeMap } from "../props";

export function Database({ iconSize = "xl-thin", ...props }: IconProps) {
  const { iconSize: pixelSize, strokeWidth } = sizeMap[iconSize];
  return (
    <svg
      height={pixelSize}
      width={pixelSize}
      viewBox="0 0 18 18"
      xmlns="http://www.w3.org/2000/svg"
      {...props}
    >
      <g>
        <ellipse
          cx="9"
          cy="4.25"
          rx="6.25"
          ry="2.25"
          fill="none"
          stroke="currentColor"
          strokeLinecap="round"
          strokeLinejoin="round"
          strokeWidth={strokeWidth}
        />
        <path
          d="M2.75,4.25V13.75c0,1.243,2.798,2.25,6.25,2.25s6.25-1.007,6.25-2.25V4.25"
          fill="none"
          stroke="currentColor"
          strokeLinecap="round"
          strokeLinejoin="round"
          strokeWidth={strokeWidth}
        />
        <path
          d="M2.75,9c0,1.243,2.798,2.25,6.25,2.25s6.25-1.007,6.25-2.25"
          fill="none"
          stroke="currentColor"
          strokeLinecap="round"
          strokeLinejoin="round"
          strokeWidth={strokeWidth}
        />
      </g>
    </svg>
  );
}
