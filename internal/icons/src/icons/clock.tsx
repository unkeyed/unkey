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

export const Clock: React.FC<IconProps> = (props) => {
  return (
    <svg {...props} height="18" width="18" viewBox="0 0 18 18" xmlns="http://www.w3.org/2000/svg">
      <g fill="currentColor">
        <polyline
          fill="none"
          points="10 6.5 10 10 12.5 12.5"
          stroke="currentColor"
          strokeLinecap="round"
          strokeLinejoin="round"
          strokeWidth="1.5"
        />
        <circle
          cx="10"
          cy="10"
          fill="none"
          r="7"
          stroke="currentColor"
          strokeLinecap="round"
          strokeLinejoin="round"
          strokeWidth="1.5"
        />
      </g>
    </svg>
  );
};
