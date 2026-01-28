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

export function ArrowUpRight({ iconSize = "xl-thin", ...props }: IconProps) {
  const { iconSize: pixelSize, strokeWidth } = sizeMap[iconSize];
  return (
    <svg
      height={pixelSize}
      width={pixelSize}
      viewBox="0 0 18 18"
      xmlns="http://www.w3.org/2000/svg"
      {...props}
    >
      <g fill="currentColor" strokeLinecap="round" strokeLinejoin="round">
        <path
          d="M3.75 15a0.75 0.75 0 0 1-0.53-1.28l10.25-10.25a0.75 0.75 0 1 1 1.06 1.06l-10.25 10.25a0.75 0.75 0 0 1-0.53 0.22z"
          fill="currentColor"
          opacity="1.000"
          strokeWidth={strokeWidth}
        />
        <path
          d="M14.25 10.51a0.75 0.75 0 0 1-0.75-0.75v-5.26h-5.26a0.75 0.75 0 0 1 0-1.5h6.01a0.75 0.75 0 0 1 0.75 0.75v6.01a0.75 0.75 0 0 1-0.75 0.75z"
          fill="currentColor"
          strokeWidth={strokeWidth}
        />
      </g>
    </svg>
  );
}
