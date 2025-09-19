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
export const Pulse: React.FC<IconProps> = ({
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
          d="M0.76 9.12h3.8l2.28-6.08 4.56 12.16 2.28-6.08h3.8"
          fill="none"
          stroke="currentColor"
          strokeLinecap="round"
          strokeMiterlimit="10"
          strokeWidth={strokeWidth}
        />
      </g>
    </svg>
  );
};
