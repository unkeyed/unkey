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

export const FolderCloud: React.FC<IconProps> = ({ iconSize = "xl-thin", filled, ...props }) => {
  const { iconSize: pixelSize, strokeWidth } = sizeMap[iconSize];
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
          d="M6 15h-3a1.5 1.5 0 0 1-1.5-1.5v-9.75a1.5 1.5 0 0 1 1.5-1.5h4.5l2.25 2.25h5.25a1.5 1.5 0 0 1 1.5 1.5v0.38"
          fill={filled ? "currentColor" : "none"}
          stroke="currentColor"
          strokeLinecap="square"
          strokeMiterlimit="10"
          strokeWidth={strokeWidth}
        />
        <path
          d="M13.95 9c-1.62 0-2.99 1.2-3.26 2.83-1.06 0.13-1.81 1.12-1.68 2.19 0.12 0.98 0.93 1.72 1.91 1.73h3.03c1.82 0 3.3-1.51 3.3-3.38s-1.48-3.38-3.3-3.37z"
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
