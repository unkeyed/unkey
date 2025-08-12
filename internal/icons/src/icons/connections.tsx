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

export const Connections: React.FC<IconProps> = ({ size = "xl-thin", ...props }) => {
  const { size: pixelSize, strokeWidth } = sizeMap[size];

  return (
    <svg
      height={pixelSize}
      width={pixelSize}
      {...props}
      viewBox="0 0 18 18"
      xmlns="http://www.w3.org/2000/svg"
    >
      <g fill="currentColor" strokeLinecap="square" strokeLinejoin="miter" strokeMiterlimit="10">
        <path
          d="M9 23L23 9"
          fill="none"
          stroke="currentColor"
          strokeLinecap="butt"
          strokeWidth={strokeWidth}
        />
        <path
          d="M9 9L23 23"
          fill="none"
          stroke="currentColor"
          strokeLinecap="butt"
          strokeWidth={strokeWidth}
        />
        <path
          d="M30.1421 16L16 1.85785L1.85786 16L16 30.1421L30.1421 16Z"
          fill="none"
          stroke="currentColor"
          strokeWidth={strokeWidth}
        />
      </g>
    </svg>
  );
};
