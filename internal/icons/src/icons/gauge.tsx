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
export const Gauge: React.FC<IconProps> = (props) => {
  return (
    <svg {...props} height="18" width="18" viewBox="0 0 18 18" xmlns="http://www.w3.org/2000/svg">
      <g fill="#212121">
        <path
          d="M6,2.398c.914-.416,1.93-.648,3-.648,4.004,0,7.25,3.246,7.25,7.25s-3.246,7.25-7.25,7.25S1.75,13.004,1.75,9c0-1.07,.232-2.086,.648-3"
          fill="none"
          stroke="#212121"
          strokeLinecap="round"
          strokeLinejoin="round"
          strokeWidth="1.5"
        />
        <circle
          cx="9"
          cy="9"
          fill="#212121"
          r="1"
          stroke="#212121"
          strokeLinecap="round"
          strokeLinejoin="round"
          strokeWidth="1.5"
        />
        <line
          fill="none"
          stroke="#212121"
          strokeLinecap="round"
          strokeLinejoin="round"
          strokeWidth="1.5"
          x1="8.293"
          x2="3.883"
          y1="8.293"
          y2="3.883"
        />
      </g>
    </svg>
  );
};
