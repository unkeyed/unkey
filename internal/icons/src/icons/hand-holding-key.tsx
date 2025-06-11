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

export const HandHoldingKey: React.FC<IconProps> = ({ size = "xl-thin", ...props }) => {
  const { size: pixelSize, strokeWidth } = sizeMap[size];

  return (
    <svg
      height={pixelSize}
      width={pixelSize}
      {...props}
      viewBox="0 0 18 18"
      xmlns="http://www.w3.org/2000/svg"
    >
      <g fill="currentColor">
        <circle
          cx="7.25"
          cy="3.75"
          fill="none"
          r="2"
          stroke="currentColor"
          strokeLinecap="round"
          strokeLinejoin="round"
          strokeWidth={strokeWidth}
        />
        <line
          fill="none"
          stroke="currentColor"
          strokeLinecap="round"
          strokeLinejoin="round"
          strokeWidth={strokeWidth}
          x1="9.25"
          x2="16.25"
          y1="3.75"
          y2="3.75"
        />
        <line
          fill="none"
          stroke="currentColor"
          strokeLinecap="round"
          strokeLinejoin="round"
          strokeWidth={strokeWidth}
          x1="15"
          x2="15"
          y1="3.75"
          y2="5.75"
        />
        <line
          fill="none"
          stroke="currentColor"
          strokeLinecap="round"
          strokeLinejoin="round"
          strokeWidth={strokeWidth}
          x1="12.75"
          x2="12.75"
          y1="3.75"
          y2="5.75"
        />
        <path
          d="M10.437,12.465l4.817-2.174c.712-.321,1.549-.005,1.871,.707h0c.321,.712,.005,1.549-.707,1.871l-6.791,3.065c-.676,.305-1.441,.349-2.148,.123l-4.478-1.432"
          fill="none"
          stroke="currentColor"
          strokeLinecap="round"
          strokeLinejoin="round"
          strokeWidth={strokeWidth}
        />
        <path
          d="M7.245,11.598l1.971,.894c.711,.323,1.549,.008,1.872-.703h0c.323-.711,.008-1.549-.703-1.872l-2.777-1.268c-2.296-1.023-4.233,.539-4.608,2.352"
          fill="none"
          stroke="currentColor"
          strokeLinecap="round"
          strokeLinejoin="round"
          strokeWidth={strokeWidth}
        />
        <path
          d="M.75,9.25h1.25c.552,0,1,.448,1,1v5c0,.552-.448,1-1,1H.75"
          fill="none"
          stroke="currentColor"
          strokeLinecap="round"
          strokeLinejoin="round"
          strokeWidth={strokeWidth}
        />
      </g>
    </svg>
  );
};
