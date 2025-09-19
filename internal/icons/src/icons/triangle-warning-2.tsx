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
export const TriangleWarning2: React.FC<IconProps> = ({
  iconsize = "xl-thin",
  ...props
}) => {
  const { iconsize: pixelSize, strokeWidth } = sizeMap[iconsize];

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
          d="m9,8c0-.552.447-1,1-1s1,.448,1,1v4.5c0,.552-.447,1-1,1s-1-.448-1-1v-4.5Z"
          fill="currentColor"
          strokeWidth={strokeWidth}
        />
        <path
          d="m13.725,16h1.471c1.54,0,2.502-1.667,1.732-3l-5.196-9c-.77-1.333-2.694-1.333-3.464,0L3.072,13c-.77,1.333.192,3,1.732,3h1.471"
          fill="none"
          stroke="currentColor"
          strokeLinecap="round"
          strokeLinejoin="round"
          strokeWidth={strokeWidth}
        />
        <circle
          cx="10"
          cy="15.75"
          fill="currentColor"
          r="1.25"
          strokeWidth={strokeWidth}
        />
      </g>
    </svg>
  );
};
