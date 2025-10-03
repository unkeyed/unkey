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

export const Grid: React.FC<IconProps> = ({ iconsize = "xl-thin", ...props }) => {
  const { iconsize: pixelSize } = sizeMap[iconsize];

  return (
    <svg
      height={pixelSize}
      width={pixelSize}
      viewBox="0 0 18 18"
      xmlns="http://www.w3.org/2000/svg"
      {...props}
    >
      <g fill="currentColor">
        <rect height="6" width="6" fill="currentColor" rx="1.75" ry="1.75" x="2" y="2" />
        <rect height="6" width="6" fill="currentColor" rx="1.75" ry="1.75" x="10" y="2" />
        <rect height="6" width="6" fill="currentColor" rx="1.75" ry="1.75" x="2" y="10" />
        <rect height="6" width="6" fill="currentColor" rx="1.75" ry="1.75" x="10" y="10" />
      </g>
    </svg>
  );
};
