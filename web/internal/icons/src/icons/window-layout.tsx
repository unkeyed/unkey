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

import { type IconProps, sizeMap } from "../props";

export function WindowLayout({ iconSize = "md-regular", ...props }: IconProps) {
  const { iconSize: pixelSize, strokeWidth } = sizeMap[iconSize];
  return (
    <svg
      height={pixelSize}
      width={pixelSize}
      {...props}
      viewBox="0 0 18 18"
      xmlns="http://www.w3.org/2000/svg"
    >
      <g fill="none" stroke="currentColor" strokeLinecap="round" strokeLinejoin="round">
        <line x1="6.25" y1="7.75" x2="6.25" y2="15.25" strokeWidth={strokeWidth} />
        <line x1="1.75" y1="7.75" x2="16.25" y2="7.75" strokeWidth={strokeWidth} />
        <rect
          x="1.75"
          y="2.75"
          width="14.5"
          height="12.5"
          rx="2"
          ry="2"
          strokeWidth={strokeWidth}
        />
        <circle cx="4.25" cy="5.25" r=".75" fill="currentColor" stroke="none" />
        <circle cx="6.75" cy="5.25" r=".75" fill="currentColor" stroke="none" />
      </g>
    </svg>
  );
}
