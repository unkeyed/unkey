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

export function Nodes2({ iconSize = "xl-thin", ...props }: IconProps) {
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
          d="M5.005 4.86069L7.49551 3.38901"
          fill="none"
          stroke="currentColor"
          strokeLinecap="round"
          strokeLinejoin="round"
          strokeWidth={strokeWidth}
        />
        <path
          d="M12.995 4.86069L10.5045 3.38901"
          fill="none"
          stroke="currentColor"
          strokeLinecap="round"
          strokeLinejoin="round"
          strokeWidth={strokeWidth}
        />
        <path
          d="M5.005 13.1393L7.49551 14.611"
          fill="none"
          stroke="currentColor"
          strokeLinecap="round"
          strokeLinejoin="round"
          strokeWidth={strokeWidth}
        />
        <path
          d="M12.995 13.1393L10.5045 14.611"
          fill="none"
          stroke="currentColor"
          strokeLinecap="round"
          strokeLinejoin="round"
          strokeWidth={strokeWidth}
        />
        <path
          d="M3.5 7.5V10.5"
          fill="none"
          stroke="currentColor"
          strokeLinecap="round"
          strokeLinejoin="round"
          strokeWidth={strokeWidth}
        />
        <path
          d="M14.5 7.5V10.5"
          fill="none"
          stroke="currentColor"
          strokeLinecap="round"
          strokeLinejoin="round"
          strokeWidth={strokeWidth}
        />
        <path
          d="M9 10.75C9.966 10.75 10.75 9.966 10.75 9C10.75 8.034 9.966 7.25 9 7.25C8.034 7.25 7.25 8.034 7.25 9C7.25 9.966 8.034 10.75 9 10.75Z"
          fill="none"
          stroke="currentColor"
          strokeLinecap="round"
          strokeLinejoin="round"
          strokeWidth={strokeWidth}
        />
        <path
          d="M3.5 7.5C4.466 7.5 5.25 6.716 5.25 5.75C5.25 4.784 4.466 4 3.5 4C2.534 4 1.75 4.784 1.75 5.75C1.75 6.716 2.534 7.5 3.5 7.5Z"
          fill="none"
          stroke="currentColor"
          strokeLinecap="round"
          strokeLinejoin="round"
          strokeWidth={strokeWidth}
        />
        <path
          d="M3.5 14C4.466 14 5.25 13.216 5.25 12.25C5.25 11.284 4.466 10.5 3.5 10.5C2.534 10.5 1.75 11.284 1.75 12.25C1.75 13.216 2.534 14 3.5 14Z"
          fill="none"
          stroke="currentColor"
          strokeLinecap="round"
          strokeLinejoin="round"
          strokeWidth={strokeWidth}
        />
        <path
          d="M14.5 7.5C13.534 7.5 12.75 6.716 12.75 5.75C12.75 4.784 13.534 4 14.5 4C15.466 4 16.25 4.784 16.25 5.75C16.25 6.716 15.466 7.5 14.5 7.5Z"
          fill="none"
          stroke="currentColor"
          strokeLinecap="round"
          strokeLinejoin="round"
          strokeWidth={strokeWidth}
        />
        <path
          d="M14.5 14C13.534 14 12.75 13.216 12.75 12.25C12.75 11.284 13.534 10.5 14.5 10.5C15.466 10.5 16.25 11.284 16.25 12.25C16.25 13.216 15.466 14 14.5 14Z"
          fill="none"
          stroke="currentColor"
          strokeLinecap="round"
          strokeLinejoin="round"
          strokeWidth={strokeWidth}
        />
        <path
          d="M9 4.25C9.966 4.25 10.75 3.466 10.75 2.5C10.75 1.534 9.966 0.75 9 0.75C8.034 0.75 7.25 1.534 7.25 2.5C7.25 3.466 8.034 4.25 9 4.25Z"
          fill="none"
          stroke="currentColor"
          strokeLinecap="round"
          strokeLinejoin="round"
          strokeWidth={strokeWidth}
        />
        <path
          d="M9 17.25C9.966 17.25 10.75 16.466 10.75 15.5C10.75 14.534 9.966 13.75 9 13.75C8.034 13.75 7.25 14.534 7.25 15.5C7.25 16.466 8.034 17.25 9 17.25Z"
          fill="none"
          stroke="currentColor"
          strokeLinecap="round"
          strokeLinejoin="round"
          strokeWidth={strokeWidth}
        />
      </g>
    </svg>
  );
}
