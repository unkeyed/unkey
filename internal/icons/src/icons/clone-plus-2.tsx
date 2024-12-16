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

import React from "react";

import type { IconProps } from "../props";
export const ClonePlus2: React.FC<IconProps> = (props) => {
  return (
    <svg
      {...props}
      height="18"
      width="18"
      viewBox="0 0 18 18"
      xmlns="http://www.w3.org/2000/svg"
    >
      <g fill="currentColor">
        <rect
          height="11"
          width="11"
          fill="none"
          rx="2"
          ry="2"
          stroke="currentColor"
          strokeLinecap="round"
          strokeLinejoin="round"
          strokeWidth="1.5"
          transform="translate(21.5 21.5) rotate(180)"
          x="5.25"
          y="5.25"
        />
        <path
          d="M3,12.605c-.733-.297-1.25-1.015-1.25-1.855V3.75c0-1.105,.895-2,2-2h7c.839,0,1.558,.517,1.855,1.25"
          fill="none"
          stroke="currentColor"
          strokeLinecap="round"
          strokeLinejoin="round"
          strokeWidth="1.5"
        />
        <line
          fill="none"
          stroke="currentColor"
          strokeLinecap="round"
          strokeLinejoin="round"
          strokeWidth="1.5"
          x1="10.75"
          x2="10.75"
          y1="13.25"
          y2="8.25"
        />
        <line
          fill="none"
          stroke="currentColor"
          strokeLinecap="round"
          strokeLinejoin="round"
          strokeWidth="1.5"
          x1="8.25"
          x2="13.25"
          y1="10.75"
          y2="10.75"
        />
      </g>
    </svg>
  );
};
