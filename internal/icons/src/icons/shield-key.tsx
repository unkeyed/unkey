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
export const ShieldKey: React.FC<IconProps> = ({ iconSize = "xl-thin", ...props }) => {
  const { iconSize: pixelSize, strokeWidth } = sizeMap[iconSize];
  return (
    <svg
      height={pixelSize}
      width={pixelSize}
      viewBox="0 0 18 18"
      xmlns="http://www.w3.org/2000/svg"
      {...props}
    >
      <g fill="currentColor">
        <circle
          cx="10"
          cy="9.25"
          fill="none"
          r="1.75"
          stroke="currentColor"
          strokeLinecap="round"
          strokeLinejoin="round"
          strokeWidth={strokeWidth}
        />
        <line
          fill="none"
          stroke="currentColor"
          strokeLinecap="round"
          strokeLinejoin="round"
          strokeWidth={strokeWidth}
          x1="12"
          x2="16.5"
          y1="9.25"
          y2="9.25"
        />
        <line
          fill="none"
          stroke="currentColor"
          strokeLinecap="round"
          strokeLinejoin="round"
          strokeWidth={strokeWidth}
          x1="15.25"
          x2="15.25"
          y1="9.25"
          y2="11.25"
        />
        <path
          d="M15.25,6.75v-2.27c0-.435-.281-.82-.695-.952l-5.25-1.68c-.198-.063-.411-.063-.61,0L3.445,3.528c-.414,.133-.695,.517-.695,.952v6.52c0,3.03,4.684,4.749,5.942,5.155,.203,.066,.413,.066,.616,0,.862-.279,3.334-1.175,4.804-2.686"
          fill="none"
          stroke="currentColor"
          strokeLinecap="round"
          strokeLinejoin="round"
          strokeWidth={strokeWidth}
        />
      </g>
    </svg>
  );
};
