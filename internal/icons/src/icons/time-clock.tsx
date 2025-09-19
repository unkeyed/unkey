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

export const TimeClock: React.FC<IconProps> = ({
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
      <g fill="currentColor" strokeLinecap="butt" strokeLinejoin="miter">
        <path
          d="M9 1.69v1.31"
          fill="none"
          stroke="currentColor"
          strokeLinecap="square"
          strokeWidth={strokeWidth}
        />
        <path
          d="M16.31 9h-1.31"
          fill="none"
          stroke="currentColor"
          strokeLinecap="square"
          strokeWidth={strokeWidth}
        />
        <path
          d="M9 16.31v-1.31"
          fill="none"
          stroke="currentColor"
          strokeLinecap="square"
          strokeWidth={strokeWidth}
        />
        <path
          d="M1.69 9h1.31"
          fill="none"
          stroke="currentColor"
          strokeLinecap="square"
          strokeWidth={strokeWidth}
        />
        <path
          d="M9 1.5a7.5 7.5 0 1 0 0 15 7.5 7.5 0 1 0 0-15z"
          fill="none"
          stroke="currentColor"
          strokeLinecap="square"
          strokeWidth={strokeWidth}
        />
        <path
          d="M6 5.25l3 3.75h3"
          fill="none"
          stroke="currentColor"
          strokeLinecap="square"
          strokeWidth={strokeWidth}
        />
      </g>
    </svg>
  );
};
