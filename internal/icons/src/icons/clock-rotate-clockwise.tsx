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

export const ClockRotateClockwise: React.FC<IconProps> = ({ iconsize, filled, ...props }) => {
  const { iconsize: pixelSize, strokeWidth } = sizeMap[iconsize || "md-regular"];

  return (
    <svg
      height={pixelSize}
      width={pixelSize}
      viewBox="0 0 18 18"
      xmlns="http://www.w3.org/2000/svg"
      {...props}
    >
      <g fill="currentColor" strokeLinecap="round" strokeLinejoin="round">
        <path
          d="M1.5 9c0-4.14 3.36-7.5 7.5-7.5s7.5 3.36 7.5 7.5-3.36 7.5-7.5 7.5a7.5 7.5 0 0 1-6.31-3.45"
          fill="none"
          stroke="currentColor"
          strokeLinecap="round"
          strokeMiterlimit="10"
          strokeWidth={strokeWidth}
        />
        <path
          d="M9 4.5v4.5l3 2.25"
          fill="none"
          stroke="currentColor"
          strokeLinecap="round"
          strokeMiterlimit="10"
          strokeWidth={strokeWidth}
        />
        <path
          d="M2.25 16.5v-3.75h3.75"
          fill="none"
          stroke="currentColor"
          strokeLinecap="round"
          strokeMiterlimit="10"
          strokeWidth={strokeWidth}
        />
      </g>
    </svg>
    // <svg
    //   {...props}
    //   height={pixelSize}
    //   width={pixelSize}
    //   viewBox="0 0 18 18"
    //   xmlns="http://www.w3.org/2000/svg"
    // >
    //   <g fill="currentColor">
    //     <polyline
    //       fill="none"
    //       points="10 7 10 10 12 12"
    //       stroke="currentColor"
    //       strokeLinecap="round"
    //       strokeLinejoin="round"
    //       strokeWidth={strokeWidth}
    //     />
    //     <polygon
    //       fill="currentColor"
    //       points="4.367 16.956 3.771 13.202 7.516 13.855 4.367 16.956"
    //       stroke="currentColor"
    //       strokeLinecap="round"
    //       strokeLinejoin="round"
    //       strokeWidth={strokeWidth}
    //     />
    //     <path
    //       d="m5,14.899c1.271,1.297,3.041,2.101,5,2.101,3.866,0,7-3.134,7-7s-3.134-7-7-7c-2.792,0-5.203,1.635-6.326,4"
    //       fill="none"
    //       stroke="currentColor"
    //       strokeLinecap="round"
    //       strokeLinejoin="round"
    //       strokeWidth={strokeWidth}
    //     />
    //   </g>
    // </svg>
  );
};
