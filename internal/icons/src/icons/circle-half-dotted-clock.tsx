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

export const CircleHalfDottedClock: React.FC<IconProps> = (props) => {
  return (
    <svg {...props} height="18" width="18" viewBox="0 0 18 18" xmlns="http://www.w3.org/2000/svg">
      <g fill="currentColor">
        <polyline
          fill="none"
          points="10 7 10 10 12 12"
          stroke="currentColor"
          strokeLinecap="round"
          strokeLinejoin="round"
          strokeWidth="1.5"
        />
        <path
          d="m10,3c3.866,0,7,3.134,7,7s-3.134,7-7,7"
          fill="none"
          stroke="currentColor"
          strokeLinecap="round"
          strokeLinejoin="round"
          strokeWidth="1.5"
        />
        <circle cx="6.5" cy="16.062" fill="currentColor" r="1" strokeWidth="0" />
        <circle cx="3.938" cy="13.5" fill="currentColor" r="1" strokeWidth="0" />
        <circle cx="3" cy="10" fill="currentColor" r="1" strokeWidth="0" />
        <circle cx="3.938" cy="6.5" fill="currentColor" r="1" strokeWidth="0" />
        <circle cx="6.5" cy="3.938" fill="currentColor" r="1" strokeWidth="0" />
      </g>
    </svg>
  );
};
