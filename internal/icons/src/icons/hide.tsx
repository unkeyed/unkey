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

export const Hide: React.FC<IconProps> = ({ size = "xl-thin", filled, ...props }) => {
  const { size: pixelSize, strokeWidth } = sizeMap[size];

  return (
    <svg
      height={pixelSize}
      width={pixelSize}
      {...props}
      viewBox="0 0 18 18"
      xmlns="http://www.w3.org/2000/svg"
    >
      <g fill="currentColor" strokeLinecap="round" strokeLinejoin="round">
        <path
          d="M4.55 13.45c-2.42-1.78-3.71-4.45-3.71-4.45s2.9-6 8.16-6c1.75 0 3.24 0.67 4.45 1.55"
          fill="none"
          stroke="currentColor"
          strokeMiterlimit="10"
          strokeWidth={strokeWidth}
        />
        <path
          d="M15.6 6.64c1.02 1.26 1.55 2.36 1.56 2.36s-2.9 6-8.16 6a6.82 6.82 0 0 1-1.57-0.18"
          fill="none"
          stroke="currentColor"
          strokeLinecap="round"
          strokeMiterlimit="10"
          strokeWidth={strokeWidth}
        />
        <path
          d="M9 6.75a2.25 2.25 0 1 0 0 4.5 2.25 2.25 0 1 0 0-4.5z"
          fill="currentColor"
          stroke="currentColor"
          strokeLinecap="round"
          strokeMiterlimit="10"
          strokeWidth={strokeWidth}
        />
        <path
          d="M16.5 1.5l-15 15"
          fill="none"
          stroke="currentColor"
          strokeLinecap="round"
          strokeMiterlimit="10"
          strokeWidth={strokeWidth}
        />
      </g>
    </svg>
  );
};
