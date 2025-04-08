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

export const ChartUsage: React.FC<IconProps> = (props) => {
  return (
    <svg height="18" width="18" {...props} viewBox="0 0 18 18" xmlns="http://www.w3.org/2000/svg">
      <g fill="currentColor">
        <path
          d="M15.602,6c.416,.914,.648,1.93,.648,3,0,2.066-.864,3.929-2.25,5.25"
          fill="none"
          stroke="currentColor"
          strokeLinecap="round"
          strokeLinejoin="round"
          strokeWidth="1.5"
        />
        <path
          d="M4,14.25c-1.386-1.321-2.25-3.184-2.25-5.25C1.75,4.996,4.996,1.75,9,1.75c1.938,0,3.699,.761,5,2"
          fill="none"
          stroke="currentColor"
          strokeLinecap="round"
          strokeLinejoin="round"
          strokeWidth="1.5"
        />
        <circle cx="7" cy="7" fill="currentColor" r="1" stroke="none" />
        <circle cx="11" cy="11" fill="currentColor" r="1" stroke="none" />
        <line
          fill="none"
          stroke="currentColor"
          strokeLinecap="round"
          strokeLinejoin="round"
          strokeWidth="1.5"
          x1="7"
          x2="11"
          y1="11.25"
          y2="6.75"
        />
      </g>
    </svg>
  );
};
