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

export const ExternalLink: React.FC<IconProps> = ({
  iconSize = "md-regular",
  filled,
  ...props
}) => {
  const { iconSize: pixelSize, strokeWidth } = sizeMap[iconSize];
  return (
    <svg
      height={pixelSize}
      width={pixelSize}
      {...props}
      viewBox="0 0 24 24"
      xmlns="http://www.w3.org/2000/svg"
    >
      <g
        fill={filled ? "currentColor" : "currentColor"}
        strokeLinecap="round"
        strokeLinejoin="round"
        transform="translate(0.25 0.25)"
      >
        <path
          d="M4.49998 21.5L14.9999 11L14.5 11.5"
          fill="none"
          stroke="currentColor"
          strokeWidth={strokeWidth}
        />
        <path d="M9 11L15 11L15 17" fill="none" stroke="currentColor" strokeWidth={strokeWidth} />
        <path
          d="M11 22L18 22C19.1046 22 20 21.1046 20 20L20 4C20 2.89543 19.1046 2 18 2L6 2C4.89543 2 4 2.89543 4 4L4 15"
          fill="none"
          stroke="currentColor"
          strokeWidth={strokeWidth}
        />
      </g>
    </svg>
  );
};
