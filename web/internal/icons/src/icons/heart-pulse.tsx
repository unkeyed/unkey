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

export function HeartPulse({ iconSize = "xl-thin", ...props }: IconProps) {
  const { iconSize: pixelSize, strokeWidth } = sizeMap[iconSize];

  return (
    <svg
      height={pixelSize}
      width={pixelSize}
      viewBox="0 0 18 18"
      xmlns="http://www.w3.org/2000/svg"
      {...props}
    >
      <g fill="currentColor">
        <path
          d="M15.663 8.75C15.877 8.083 16 7.369 16 6.609C16.008 4.489 14.296 2.763 12.174 2.75C10.897 2.766 9.70998 3.41 9.00098 4.47C8.29098 3.41 7.10399 2.766 5.82799 2.75C3.70499 2.763 1.99398 4.489 2.00198 6.609C2.00198 7.563 2.19598 8.444 2.51898 9.25"
          fill="none"
          stroke="currentColor"
          strokeLinecap="round"
          strokeLinejoin="round"
          strokeWidth={strokeWidth}
        />
        <path
          d="M4.51349 12.2512C5.98729 13.7937 7.71499 14.7973 8.52999 15.222C8.82699 15.377 9.17399 15.377 9.47099 15.222C10.3714 14.7525 12.3843 13.5774 13.9393 11.75"
          fill="none"
          stroke="currentColor"
          strokeLinecap="round"
          strokeLinejoin="round"
          strokeWidth={strokeWidth}
        />
        <path
          d="M17.25 8.75H12L10.5 11.751L7.25 6.25L5.75 9.251H0.75"
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
