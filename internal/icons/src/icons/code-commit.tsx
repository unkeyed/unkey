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

export const CodeCommit: React.FC<IconProps> = ({ size = "xl-thin", ...props }) => {
  const { size: pixelSize, strokeWidth } = sizeMap[size];

  return (
    <svg
      height={pixelSize}
      width={pixelSize}
      {...props}
      viewBox="0 0 32 32"
      xmlns="http://www.w3.org/2000/svg"
    >
      <g fill="currentColor" strokeLinecap="square" strokeLinejoin="miter" strokeMiterlimit="10">
        <path d="M16 27V31" fill="none" stroke="currentColor" strokeWidth={strokeWidth} />
        <path
          d="M16 23C19.866 23 23 19.866 23 16C23 12.134 19.866 9 16 9C12.134 9 9 12.134 9 16C9 19.866 12.134 23 16 23Z"
          fill="none"
          stroke="currentColor"
          strokeWidth={strokeWidth}
        />
        <path d="M16 1.00012V9.00012" fill="none" stroke="currentColor" strokeWidth={strokeWidth} />
      </g>
    </svg>
  );
};
