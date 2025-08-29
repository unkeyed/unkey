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

export const Nut: React.FC<IconProps> = ({ size = "xl-thin", ...props }) => {
  const { size: pixelSize, strokeWidth } = sizeMap[size];

  return (
    <svg
      height={pixelSize}
      width={pixelSize}
      {...props}
      viewBox="0 0 32 32"
      xmlns="http://www.w3.org/2000/svg"
    >
      <g
        fill="currentColor"
        strokeLinecap="square"
        strokeLinejoin="miter"
        strokeMiterlimit="10"
        strokeWidth={strokeWidth}
      >
        <polygon
          fill="none"
          points="9 29 1 16 9 3 23 3 31 16 23 29 9 29"
          stroke="currentColor"
          strokeWidth={strokeWidth}
        />
        <circle cx="16" cy="16" fill="none" r="6" stroke="currentColor" strokeWidth={strokeWidth} />
      </g>
    </svg>
  );
};
