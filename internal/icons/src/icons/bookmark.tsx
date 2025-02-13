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

export const Bookmark: React.FC<IconProps> = ({ size, filled, ...props }) => {
  const { size: pixelSize, strokeWidth } = sizeMap[size || "md-regular"];
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
          d="M15 16.5l-6-3.75-6 3.75v-13.5a1.5 1.5 0 0 1 1.5-1.5h9a1.5 1.5 0 0 1 1.5 1.5v13.5z"
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
