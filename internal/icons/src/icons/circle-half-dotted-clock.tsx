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

export const CircleHalfDottedClock: React.FC<IconProps> = ({ iconsize = "xl-thin", ...props }) => {
  const { iconsize: pixelSize, strokeWidth } = sizeMap[iconsize];

  return (
    <svg
      height={pixelSize}
      width={pixelSize}
      viewBox="0 0 18 18"
      xmlns="http://www.w3.org/2000/svg"
      {...props}
    >
      <g fill="currentColor">
        <polyline
          fill="none"
          points="9 4.75 9 9 12.25 11.25"
          stroke="currentColor"
          strokeLinecap="round"
          strokeLinejoin="round"
          strokeWidth={strokeWidth}
        />
        <path
          d="M9,1.75c4.004,0,7.25,3.246,7.25,7.25s-3.246,7.25-7.25,7.25"
          fill="none"
          stroke="currentColor"
          strokeLinecap="round"
          strokeLinejoin="round"
          strokeWidth={strokeWidth}
        />
        <circle cx="3.873" cy="14.127" fill="currentColor" r=".75" stroke="none" />
        <circle cx="1.75" cy="9" fill="currentColor" r=".75" stroke="none" />
        <circle cx="3.873" cy="3.873" fill="currentColor" r=".75" stroke="none" />
        <circle cx="6.226" cy="15.698" fill="currentColor" r=".75" stroke="none" />
        <circle cx="2.302" cy="11.774" fill="currentColor" r=".75" stroke="none" />
        <circle cx="2.302" cy="6.226" fill="currentColor" r=".75" stroke="none" />
        <circle cx="6.226" cy="2.302" fill="currentColor" r=".75" stroke="none" />
      </g>
    </svg>
  );
};
