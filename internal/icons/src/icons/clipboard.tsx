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

export const Clipboard: React.FC<IconProps> = (props) => {
  return (
    <svg {...props} height="18" width="18" viewBox="0 0 18 18" xmlns="http://www.w3.org/2000/svg">
      <g fill="currentColor">
        <path
          d="M6.25,2.75h-1c-1.105,0-2,.895-2,2V14.25c0,1.105,.895,2,2,2h7.5c1.105,0,2-.895,2-2V4.75c0-1.105-.895-2-2-2h-1"
          fill="none"
          stroke="currentColor"
          strokeLinecap="round"
          strokeLinejoin="round"
          strokeWidth="1.5"
        />
        <rect
          height="3"
          width="5.5"
          fill="none"
          rx="1"
          ry="1"
          stroke="currentColor"
          strokeLinecap="round"
          strokeLinejoin="round"
          strokeWidth="1.5"
          x="6.25"
          y="1.25"
        />
      </g>
    </svg>
  );
};
