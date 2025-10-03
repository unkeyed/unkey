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

export const CaretExpandY: React.FC<IconProps> = ({ iconsize = "xl-thin", ...props }) => {
  const { iconsize: pixelSize, strokeWidth } = sizeMap[iconsize];
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
          d="M8.586,3.001l-2.348,3.468c-.225,.332,.013,.78,.414,.78h4.696c.401,0,.639-.448,.414-.78l-2.348-3.468c-.198-.293-.63-.293-.828,0Z"
          fill="currentColor"
          stroke="currentColor"
          strokeLinecap="round"
          strokeLinejoin="round"
          strokeWidth={strokeWidth}
        />
        <path
          d="M8.586,14.999l-2.348-3.468c-.225-.332,.013-.78,.414-.78h4.696c.401,0,.639,.448,.414,.78l-2.348,3.468c-.198,.293-.63,.293-.828,0Z"
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
