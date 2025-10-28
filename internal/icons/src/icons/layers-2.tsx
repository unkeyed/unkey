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

export const Layers2: React.FC<IconProps> = ({ iconSize = "xl-regular", ...props }) => {
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
          d="m3.034,12.231c-.111.475.072,1.01.555,1.286l5.83,3.332c.36.206.801.206,1.161,0l5.83-3.332c.483-.276.667-.811.555-1.286"
          fill="none"
          stroke="currentColor"
          strokeLinecap="round"
          strokeLinejoin="round"
          strokeWidth={strokeWidth}
        />
        <path
          d="m10.58,3.154l5.83,3.332c.786.449.786,1.582,0,2.031l-5.83,3.332c-.36.205-.801.205-1.161,0l-5.83-3.332c-.786-.449-.786-1.582,0-2.031l5.83-3.332c.36-.205.801-.205,1.161,0Z"
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
