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

export const Key: React.FC<IconProps> = ({
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
        <path
          d="M15.747,2.076l-2.847,.177-5.891,5.891c-.324-.084-.658-.144-1.009-.144-2.209,0-4,1.791-4,4s1.791,4,4,4,4-1.791,4-4c0-.362-.064-.707-.154-1.041l1.904-1.959v-2.25h2.25l1.753-1.645-.006-3.029Z"
          fill="none"
          stroke="currentColor"
          strokeLinecap="round"
          strokeLinejoin="round"
          strokeWidth={strokeWidth}
        />
        <circle cx="5.5" cy="12.5" fill="currentColor" r="1" stroke="none" />
      </g>
    </svg>
  );
};
