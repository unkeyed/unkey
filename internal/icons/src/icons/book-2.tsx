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

export const Book2: React.FC<IconProps> = ({ iconsize = "xl-thin", ...props }) => {
  const { iconsize: pixelSize, strokeWidth } = sizeMap[iconsize];
  return (
    <svg
      height={pixelSize}
      width={pixelSize}
      viewBox="0 0 18 18"
      xmlns="http://www.w3.org/2000/svg"
      {...props}
    >
      <g fill="currentColor" strokeLinecap="butt" strokeLinejoin="miter">
        <path
          d="M2.63 14.63a2.25 2.25 0 0 0 2.25 2.24h10.5"
          fill="none"
          stroke="currentColor"
          strokeLinecap="butt"
          strokeLinejoin="miter"
          strokeWidth={strokeWidth}
        />
        <path
          d="M2.63 14.63v-11.26a2.25 2.25 0 0 1 2.25-2.25h8.25a2.25 2.25 0 0 1 2.25 2.25v9h-10.5a2.25 2.25 0 0 0-2.25 2.25z"
          fill="none"
          stroke="currentColor"
          strokeLinecap="butt"
          strokeLinejoin="miter"
          strokeWidth={strokeWidth}
        />
        <path
          d="M10.88 6.38h-3.76"
          fill="none"
          stroke="currentColor"
          strokeLinecap="butt"
          strokeLinejoin="miter"
          strokeWidth={strokeWidth}
        />
      </g>
    </svg>
  );
};
