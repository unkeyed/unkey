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

// biome-ignore lint: React in this context is used throughout, so biome will change to types because no APIs are used even though React is needed.
import React from "react";
import type { IconProps } from "../props";

export const ChevronRight: React.FC<IconProps> = (props: IconProps) => {
  return (
    <svg {...props} height="18" width="18" viewBox="0 0 18 18" xmlns="http://www.w3.org/2000/svg">
      <g fill="currentColor">
        <path
          d="M6.5 2.75L12.75 9L6.5 15.25"
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
