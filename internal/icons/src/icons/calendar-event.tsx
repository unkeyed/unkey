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

import { type IconProps, sizeMap } from "../props";

export const CalendarEvent: React.FC<IconProps> = ({ size, filled, ...props }) => {
  const { size: pixelSize, strokeWidth } = sizeMap[size || "md-regular"];

  return (
    <svg
      height={pixelSize}
      width={pixelSize}
      {...props}
      viewBox="0 0 18 18"
      xmlns="http://www.w3.org/2000/svg"
    >
      <g fill="currentColor">
        <path
          d="M12 10.5h1.5v1.5h-1.5z"
          fill={filled ? "currentColor" : "none"}
          stroke="currentColor"
          strokeLinecap="square"
          strokeMiterlimit="10"
          strokeWidth={strokeWidth}
        />
        <path
          d="M1.5 6.75h15"
          fill={filled ? "currentColor" : "none"}
          stroke="currentColor"
          strokeLinecap="square"
          strokeMiterlimit="10"
          strokeWidth={strokeWidth}
        />
        <path
          d="M3 3h12q1.5 0 1.5 1.5v9q0 1.5-1.5 1.5h-12q-1.5 0-1.5-1.5v-9q0-1.5 1.5-1.5z"
          fill={filled ? "currentColor" : "none"}
          stroke="currentColor"
          strokeLinecap="square"
          strokeMiterlimit="10"
          strokeWidth={strokeWidth}
        />
        <path
          d="M5.25 0.75v2.25"
          fill={filled ? "currentColor" : "none"}
          stroke="currentColor"
          strokeLinecap="square"
          strokeMiterlimit="10"
          strokeWidth={strokeWidth}
        />
        <path
          d="M12.75 0.75v2.25"
          fill={filled ? "currentColor" : "none"}
          stroke="currentColor"
          strokeLinecap="square"
          strokeMiterlimit="10"
          strokeWidth={strokeWidth}
        />
      </g>
    </svg>
  );
};

export default CalendarEvent;
