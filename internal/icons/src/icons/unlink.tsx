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
// Uses Link4 with center removed
export const Unlink: React.FC<IconProps> = ({
  size = "md-regular",
  ...props
}) => {
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
          d="M9 18c-3.31 0-6-2.69-6-6v-0.75a1.13 1.13 0 0 1 2.25 0v0.75c0 2.07 1.68 3.75 3.75 3.75s3.75-1.68 3.75-3.75v-0.75a1.13 1.13 0 0 1 2.25 0v0.75c0 3.31-2.69 6-6 6z"
          fill="currentColor"
          strokeWidth={strokeWidth}
        />
        <path
          d="M13.88 7.88a1.13 1.13 0 0 1-1.13-1.13v-0.75c0-2.07-1.68-3.75-3.75-3.75s-3.75 1.68-3.75 3.75v0.75a1.13 1.13 0 0 1-2.25 0v-0.75c0-3.31 2.69-6 6-6s6 2.69 6 6v0.75a1.13 1.13 0 0 1-1.13 1.13z"
          fill="currentColor"
          strokeWidth={strokeWidth}
        />
        {/*<path
          d="M9 13.5a1.13 1.13 0 0 1-1.13-1.13v-6.75a1.13 1.13 0 0 1 2.25 0v6.75a1.13 1.13 0 0 1-1.12 1.13z"
          fill="currentColor"
          strokeWidth={strokeWidth}
        />*/}
      </g>
    </svg>
  );
};
