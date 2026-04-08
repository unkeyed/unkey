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

export function Minimize({ iconSize = "xl-thin", ...props }: IconProps) {
  const { iconSize: pixelSize, strokeWidth } = sizeMap[iconSize];

  return (
    <svg
      xmlns="http://www.w3.org/2000/svg"
      x="0px"
      y="0px"
      width={pixelSize}
      height={pixelSize}
      viewBox="0 0 18 18"
      {...props}
    >
      <path
        d="M5.75 11.75H3.75C2.645 11.75 1.75 10.855 1.75 9.75V4.75C1.75 3.645 2.645 2.75 3.75 2.75H12.25C13.355 2.75 14.25 3.645 14.25 4.75V7.75"
        stroke="currentColor"
        strokeWidth={strokeWidth}
        strokeLinecap="round"
        strokeLinejoin="round"
        fill="none"
      />
      <path
        d="M14.75 10.75H10.25C9.422 10.75 8.75 11.422 8.75 12.25V13.75C8.75 14.578 9.422 15.25 10.25 15.25H14.75C15.578 15.25 16.25 14.578 16.25 13.75V12.25C16.25 11.422 15.578 10.75 14.75 10.75Z"
        stroke="currentColor"
        strokeWidth={strokeWidth}
        strokeLinecap="round"
        strokeLinejoin="round"
        fill="none"
      />
      <path
        d="M7.25 5.25V8.25H4.25"
        stroke="currentColor"
        strokeWidth={strokeWidth}
        strokeLinecap="round"
        strokeLinejoin="round"
        data-color="color-2"
        fill="none"
      />
      <path
        d="M7.25 8.25L4.25 5.25"
        stroke="currentColor"
        strokeWidth={strokeWidth}
        strokeLinecap="round"
        strokeLinejoin="round"
        data-color="color-2"
        fill="none"
      />
    </svg>
  );
}
