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

import type { IconProps } from "../props";
export const TriangleWarning2: React.FC<IconProps> = (props) => {
  return (
    <svg {...props} height="12" width="12" viewBox="0 0 12 12" xmlns="http://www.w3.org/2000/svg">
      <g fill="currentColor">
        <circle cx="6" cy="10.125" fill="currentColor" r=".875" strokeWidth="0" />
        <line
          fill="none"
          stroke="currentColor"
          strokeLinecap="round"
          strokeLinejoin="round"
          strokeWidth="1.5"
          x1="6"
          x2="6"
          y1="4.75"
          y2="7.75"
        />
        <path
          d="m8.625,10.25h1.164c1.123,0,1.826-1.216,1.265-2.189L7.265,1.484c-.562-.975-1.969-.975-2.53,0L.946,8.061c-.561.973.142,2.189,1.265,2.189h1.164"
          fill="none"
          stroke="currentColor"
          strokeLinecap="round"
          strokeLinejoin="round"
          strokeWidth="1.5"
        />
      </g>
    </svg>
  );
};
