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

export const Magnifier: React.FC<IconProps> = (props) => {
  return (
    <svg {...props} height="18" width="18" viewBox="0 0 18 18" xmlns="http://www.w3.org/2000/svg">
      <g fill="currentColor">
        <line
          fill="none"
          stroke="currentColor"
          strokeLinecap="round"
          strokeLinejoin="round"
          strokeWidth="1.5"
          x1="15.25"
          x2="11.285"
          y1="15.25"
          y2="11.285"
        />
        <circle
          cx="7.75"
          cy="7.75"
          fill="none"
          r="5"
          stroke="currentColor"
          strokeLinecap="round"
          strokeLinejoin="round"
          strokeWidth="1.5"
        />
      </g>
    </svg>
  );
};
