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

import { type IconProps, sizeMap } from "../props";

export const UserSearch: React.FC<IconProps> = ({
  iconsize,
  filled,
  ...props
}) => {
  const { iconsize: pixelSize, strokeWidth } =
    sizeMap[iconsize || "md-regular"];

  return (
    <svg
      height={pixelSize}
      width={pixelSize}
      viewBox="0 0 18 18"
      xmlns="http://www.w3.org/2000/svg"
      {...props}
    >
      <g fill="currentColor">
        <path
          d="M9.12 7.22a3.04 3.04 0 1 0 0-6.08 3.04 3.04 0 0 0 0 6.08z"
          fill={filled ? "currentColor" : "none"}
          stroke="currentColor"
          strokeLinecap="square"
          strokeMiterlimit="10"
          strokeWidth={strokeWidth}
        />
        <path
          d="M9.88 9.16a7.37 7.37 0 0 0-0.76-0.04c-3.78 0-6.84 2.87-6.84 6.42 2.53 0.59 5.07 0.86 7.6 0.79"
          fill={filled ? "currentColor" : "none"}
          stroke="currentColor"
          strokeLinecap="square"
          strokeMiterlimit="10"
          strokeWidth={strokeWidth}
        />
        <path
          d="M13.11 15.58a2.85 2.85 0 1 0 0-5.7 2.85 2.85 0 0 0 0 5.7z"
          fill={filled ? "currentColor" : "none"}
          stroke="currentColor"
          strokeLinecap="square"
          strokeMiterlimit="10"
          strokeWidth={strokeWidth}
        />
        <path
          d="M16.91 16.53l-1.71-1.71 0.19 0.19"
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
