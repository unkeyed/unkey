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

export function Connections3({ iconSize = "xl-thin", ...props }: IconProps) {
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
        <rect
          height="3.889"
          width="3.889"
          fill="none"
          stroke="currentColor"
          strokeLinecap="round"
          strokeLinejoin="round"
          strokeWidth={strokeWidth}
          transform="translate(5.636 -5.121) rotate(45)"
          x="7.055"
          y="2.298"
        />
        <rect
          height="3.889"
          width="3.889"
          fill="none"
          stroke="currentColor"
          strokeLinecap="round"
          strokeLinejoin="round"
          strokeWidth={strokeWidth}
          transform="translate(-5.121 5.636) rotate(-45)"
          x="2.298"
          y="7.055"
        />
        <rect
          height="3.889"
          width="3.889"
          fill="none"
          stroke="currentColor"
          strokeLinecap="round"
          strokeLinejoin="round"
          strokeWidth={strokeWidth}
          transform="translate(5.636 29.849) rotate(-135)"
          x="7.055"
          y="11.813"
        />
        <rect
          height="3.889"
          width="3.889"
          fill="none"
          stroke="currentColor"
          strokeLinecap="round"
          strokeLinejoin="round"
          strokeWidth={strokeWidth}
          transform="translate(29.849 5.636) rotate(135)"
          x="11.813"
          y="7.055"
        />
      </g>
    </svg>
  );
}
