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

export const CCheck: React.FC<IconProps> = ({ size, filled, ...props }) => {
  const { size: pixelSize, strokeWidth } = sizeMap[size || "md-regular"];
  return (
    <svg width={pixelSize} height={pixelSize} {...props}>
      <path
        d="M9 1.5a7.5 7.5 0 1 0 0 15 7.5 7.5 0 1 0 0-15z"
        fill={filled ? "currentColor" : "none"}
        stroke="currentColor"
        strokeLinecap="square"
        strokeMiterlimit="10"
        strokeWidth={strokeWidth}
      />
      {/* other svg elements */}
    </svg>
  );
};
  return (
    <svg
      {...props}
      height={pixelSize}
      width={pixelSize}
      viewBox="0 0 18 18"
      xmlns="http://www.w3.org/2000/svg"
    >
      <g fill="currentColor" strokeLinecap="butt" strokeLinejoin="miter">
        <path
          d="M9 1.5a7.5 7.5 0 1 0 0 15 7.5 7.5 0 1 0 0-15z"
          fill="none"
          stroke="currentColor"
          strokeLinecap="square"
          strokeMiterlimit="10"
          strokeWidth={strokeWidth}
        />
        <path
          d="M5.25 9.75l2.25 2.25 5.25-6"
          fill="none"
          stroke="currentColor"
          strokeLinecap="square"
          strokeMiterlimit="10"
          strokeWidth={strokeWidth}
        />
      </g>
    </svg>
  );
};
