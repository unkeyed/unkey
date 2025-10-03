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

export const CodeBranch: React.FC<IconProps> = ({ iconsize = "xl-thin", ...props }) => {
  const { iconsize: pixelSize, strokeWidth } = sizeMap[iconsize];

  return (
    <svg
      height={pixelSize}
      width={pixelSize}
      viewBox="0 0 18 18"
      xmlns="http://www.w3.org/2000/svg"
      {...props}
    >
      <g
        fill="none"
        strokeLinecap="round"
        strokeLinejoin="round"
        strokeWidth={strokeWidth}
        stroke="currentColor"
      >
        <line x1="4.75" y1="5.75" x2="4.75" y2="12.25" />
        <path d="M13.25,5.75v1c0,1.105-.895,2-2,2H6.75c-1.105,0-2,.895-2,2" />
        <circle cx="4.75" cy="3.75" r="2" />
        <circle cx="13.25" cy="3.75" r="2" />
        <circle cx="4.75" cy="14.25" r="2" />
      </g>
    </svg>
  );
};
