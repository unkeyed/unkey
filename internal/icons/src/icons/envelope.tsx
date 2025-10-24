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

export const Envelope: React.FC<IconProps> = ({ iconsize = "md-regular", filled, ...props }) => {
  const { iconsize: pixelSize, strokeWidth } = sizeMap[iconsize];
  return (
    <svg
      height={pixelSize}
      width={pixelSize}
      {...props}
      viewBox="0 0 20 20"
      xmlns="http://www.w3.org/2000/svg"
    >
      <g fill={filled ? "currentColor" : "none"}>
        <path
          d="m3,7l6.504,3.716c.307.176.685.176.992,0l6.504-3.716"
          fill="none"
          stroke="currentColor"
          strokeLinecap="round"
          strokeLinejoin="round"
          strokeWidth={strokeWidth}
        />
        <rect
          height="12"
          width="14"
          fill="none"
          rx="3"
          ry="3"
          stroke="currentColor"
          strokeLinecap="round"
          strokeLinejoin="round"
          strokeWidth={strokeWidth}
          x="3"
          y="4"
        />
      </g>
    </svg>
  );
};
