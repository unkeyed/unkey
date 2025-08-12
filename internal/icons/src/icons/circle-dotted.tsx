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

export const CircleDotted: React.FC<IconProps> = ({ size = "xl-thin", ...props }) => {
  const { size: pixelSize, strokeWidth } = sizeMap[size];
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
        strokeLinecap="round"
        strokeLinejoin="round"
        strokeWidth={strokeWidth}
        stroke="currentColor"
      >
        <circle
          cx="3.873"
          cy="14.127"
          r="0.75"
          fill="currentColor"
          data-stroke="none"
          stroke="none"
        />
        <circle cx="1.75" cy="9" r="0.75" fill="currentColor" data-stroke="none" stroke="none" />
        <circle
          cx="3.873"
          cy="3.873"
          r="0.75"
          fill="currentColor"
          data-stroke="none"
          stroke="none"
        />
        <circle
          cx="6.226"
          cy="15.698"
          r="0.75"
          fill="currentColor"
          data-stroke="none"
          stroke="none"
        />
        <circle
          cx="2.302"
          cy="11.774"
          r="0.75"
          fill="currentColor"
          data-stroke="none"
          stroke="none"
        />
        <circle
          cx="2.302"
          cy="6.226"
          r="0.75"
          fill="currentColor"
          data-stroke="none"
          stroke="none"
        />
        <circle
          cx="6.226"
          cy="2.302"
          r="0.75"
          fill="currentColor"
          data-stroke="none"
          stroke="none"
        />
        <circle cx="9" cy="1.75" r="0.75" fill="currentColor" data-stroke="none" stroke="none" />
        <circle cx="9" cy="16.25" r="0.75" fill="currentColor" data-stroke="none" stroke="none" />
        <circle
          cx="14.127"
          cy="14.127"
          r="0.75"
          fill="currentColor"
          data-stroke="none"
          stroke="none"
        />
        <circle cx="16.25" cy="9" r="0.75" fill="currentColor" data-stroke="none" stroke="none" />
        <circle
          cx="14.127"
          cy="3.873"
          r="0.75"
          fill="currentColor"
          data-stroke="none"
          stroke="none"
        />
        <circle
          cx="11.774"
          cy="15.698"
          r="0.75"
          fill="currentColor"
          data-stroke="none"
          stroke="none"
        />
        <circle
          cx="15.698"
          cy="11.774"
          r="0.75"
          fill="currentColor"
          data-stroke="none"
          stroke="none"
        />
        <circle
          cx="15.698"
          cy="6.226"
          r="0.75"
          fill="currentColor"
          data-stroke="none"
          stroke="none"
        />
        <circle
          cx="11.774"
          cy="2.302"
          r="0.75"
          fill="currentColor"
          data-stroke="none"
          stroke="none"
        />
      </g>
    </svg>
  );
};
