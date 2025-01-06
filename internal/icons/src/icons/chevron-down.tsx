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

export const ChevronDown: React.FC<IconProps> = (props) => {
  return (
    <svg
      {...props}
      height="18"
      width="18"
      viewBox="0 0 18 18"
      xmlns="http://www.w3.org/2000/svg"
    >
      <g fill="currentColor">
        <polyline
          fill="none"
          points="15.25 6.5 9 12.75 2.75 6.5"
          stroke="currentColor"
          strokeLinecap="round"
          strokeLinejoin="round"
        />
      </g>
    </svg>
  );
};
