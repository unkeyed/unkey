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

export const Code: React.FC<IconProps> = ({
  iconSize = "xl-thin",
  ...props
}) => {
  const { iconSize: pixelSize, strokeWidth } = sizeMap[iconSize];

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
          points="6.5 13.75 1.75 9 6.5 4.25"
          stroke="currentColor"
          strokeLinecap="round"
          strokeLinejoin="round"
          strokeWidth={strokeWidth}
        />
        <polyline
          fill="none"
          points="11.5 13.75 16.25 9 11.5 4.25"
          stroke="currentColor"
          strokeLinecap="round"
          strokeLinejoin="round"
          strokeWidth={strokeWidth}
        />
      </g>
    </svg>
  );
};
