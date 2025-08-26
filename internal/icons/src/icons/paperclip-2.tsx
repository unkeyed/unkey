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

export const PaperClip2: React.FC<IconProps> = ({ size = "xl-thin", ...props }) => {
  const { size: pixelSize, strokeWidth } = sizeMap[size];

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
          d="M10.985,5.422l-4.773,4.773c-.586,.586-.586,1.536,0,2.121h0c.586,.586,1.536,.586,2.121,0l4.95-4.95c1.172-1.172,1.172-3.071,0-4.243h0c-1.172-1.172-3.071-1.172-4.243,0l-4.95,4.95c-1.757,1.757-1.757,4.607,0,6.364h0c1.757,1.757,4.607,1.757,6.364,0l4.773-4.773"
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
