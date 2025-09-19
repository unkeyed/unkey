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
import React from "react";
import { type IconProps, sizeMap } from "../props";

export const HalfDottedCirclePlay: React.FC<IconProps> = ({
  iconsize = "xl-thin",
  ...props
}) => {
  const { iconsize: pixelSize, strokeWidth } = sizeMap[iconsize];

  return (
    <svg
      height={pixelSize}
      width={pixelSize}
      viewBox="0 0 18 18"
      xmlns="http://www.w3.org/2000/svg"
      {...props}
    >
      <g
        fill="none"
        strokeLinecap="round"
        strokeLinejoin="round"
        strokeWidth={strokeWidth}
        stroke="currentColor"
      >
        <path d="M11.652,8.568l-3.651-2.129c-.333-.194-.752,.046-.752,.432v4.259c0,.386,.419,.626,.752,.432l3.651-2.129c.331-.193,.331-.671,0-.864Z" />
        <path d="M9,1.75c4.004,0,7.25,3.246,7.25,7.25s-3.246,7.25-7.25,7.25" />
        <circle
          cx="3.873"
          cy="14.127"
          r=".75"
          fill="currentColor"
          data-stroke="none"
          stroke="none"
        />
        <circle
          cx="1.75"
          cy="9"
          r=".75"
          fill="currentColor"
          data-stroke="none"
          stroke="none"
        />
        <circle
          cx="3.873"
          cy="3.873"
          r=".75"
          fill="currentColor"
          data-stroke="none"
          stroke="none"
        />
        <circle
          cx="6.226"
          cy="15.698"
          r=".75"
          fill="currentColor"
          data-stroke="none"
          stroke="none"
        />
        <circle
          cx="2.302"
          cy="11.774"
          r=".75"
          fill="currentColor"
          data-stroke="none"
          stroke="none"
        />
        <circle
          cx="2.302"
          cy="6.226"
          r=".75"
          fill="currentColor"
          data-stroke="none"
          stroke="none"
        />
        <circle
          cx="6.226"
          cy="2.302"
          r=".75"
          fill="currentColor"
          data-stroke="none"
          stroke="none"
        />
      </g>
    </svg>
  );
};
