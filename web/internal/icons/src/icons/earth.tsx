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

export const Earth: React.FC<IconProps> = ({ iconSize = "xl-thin", ...props }) => {
  const { iconSize: pixelSize, strokeWidth } = sizeMap[iconSize];

  return (
    <svg
      xmlns="http://www.w3.org/2000/svg"
      width={pixelSize}
      height={pixelSize}
      viewBox="0 0 18 18"
      {...props}
    >
      <g
        fill="none"
        stroke="currentColor"
        strokeLinecap="round"
        strokeLinejoin="round"
        strokeWidth={strokeWidth}
      >
        <path d="M5.75421 9.8474C6.90811 9.6041 8.3363 9.1472 9.9917 10.7675C12.1745 12.904 12.6914 7.9875 15.8171 11.2571" />
        <path d="M13.0008 2.9807L11.4893 2.9094C10.4948 2.8625 9.73549 3.7861 9.97369 4.7527L10.2457 5.8562C10.3051 6.0973 10.2086 6.3499 10.0036 6.4899C9.838 6.603 9.62659 6.6251 9.44119 6.5487L8.5141 6.1666C7.7892 5.8678 6.96159 5.9623 6.32269 6.4169C5.75689 6.8194 5.40529 7.4578 5.36749 8.1511L5.29651 9.4532" />
        <path d="M2.59167 5.7457C3.02027 6.8697 3.97028 8.6883 5.49658 9.6832C5.92248 9.9178 6.90028 10.6811 6.83228 11.8894C6.73908 13.5436 7.35876 13.633 8.15756 14.2274C8.56766 14.5326 8.67218 15.4704 8.61218 16.2104" />
        <path d="M9 16.25C13.004 16.25 16.25 13.0041 16.25 9C16.25 4.9959 13.004 1.75 9 1.75C4.996 1.75 1.75 4.9959 1.75 9C1.75 13.0041 4.996 16.25 9 16.25Z" />
      </g>
    </svg>
  );
};
