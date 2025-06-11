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

export const View: React.FC<IconProps> = ({ size = "xl-thin", filled, ...props }) => {
  const { size: pixelSize, strokeWidth } = sizeMap[size];

  return (
    <svg
      height={pixelSize}
      width={pixelSize}
      {...props}
      viewBox="0 0 18 18"
      xmlns="http://www.w3.org/2000/svg"
    >
      <g fill="currentColor" strokeLinecap="round" strokeLinejoin="round">
        <path
          d="M9 6.75a2.25 2.25 0 1 0 0 4.5 2.25 2.25 0 1 0 0-4.5z"
          fill="currentColor"
          stroke="currentColor"
          strokeLinecap="round"
          strokeMiterlimit="10"
          strokeWidth={strokeWidth}
        />
        <path
          d="M0.84 9s2.9-6 8.16-6 8.16 6 8.16 6-2.9 6-8.16 6-8.16-6-8.16-6z"
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
