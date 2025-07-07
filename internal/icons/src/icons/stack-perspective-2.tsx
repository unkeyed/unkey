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

export const StackPerspective2: React.FC<IconProps> = ({ size = "xl-thin", ...props }) => {
  const { size: pixelSize, strokeWidth } = sizeMap[size];

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
          d="M13.75,13.375l.691,.146c.932,.196,1.809-.515,1.809-1.468V4.619c0-.709-.497-1.322-1.191-1.468l-6.5-1.368c-.712-.15-1.388,.231-1.669,.843"
          fill="none"
          stroke="currentColor"
          strokeLinecap="round"
          strokeLinejoin="round"
          strokeWidth={strokeWidth}
        />
        <path
          d="M3.559,4.479l6.5,1.368c.694,.146,1.191,.758,1.191,1.468v7.434c0,.953-.877,1.664-1.809,1.468l-6.5-1.368c-.694-.146-1.191-.758-1.191-1.468V5.947c0-.953,.877-1.664,1.809-1.468Z"
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
