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

export const CaretDown: React.FC<IconProps> = ({ iconSize = "xl-thin", ...props }) => {
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
          d="M9.845,14.209l5.025-7.923c.422-.666-.056-1.536-.845-1.536H3.975c-.788,0-1.267,.87-.845,1.536l5.025,7.923c.393,.619,1.296,.619,1.689,0Z"
          fill="currentColor"
          stroke="currentColor"
          strokeLinecap="round"
          strokeLinejoin="round"
          strokeWidth={strokeWidth}
        />
      </g>
    </svg>
  );
};
