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

export const InputPasswordSettings: React.FC<IconProps> = ({
  iconSize = "xl-thin",
  ...props
}) => {
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
        <circle cx="5.5" cy="9" fill="currentColor" r="1" strokeWidth="0" />
        <circle cx="9" cy="9" fill="currentColor" r="1" strokeWidth="0" />
        <circle
          cx="13.75"
          cy="13.75"
          fill="none"
          r="2"
          stroke="currentColor"
          strokeLinecap="round"
          strokeLinejoin="round"
          strokeWidth={strokeWidth}
        />
        <path
          d="m16.25,8.972v-2.222c0-1.104-.895-2-2-2H3.75c-1.105,0-2,.896-2,2v4.5c0,1.104.895,2,2,2h4.31"
          fill="none"
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
          x1="13.75"
          x2="13.75"
          y1="10.5"
          y2="11.5"
        />
        <line
          fill="none"
          stroke="currentColor"
          strokeLinecap="round"
          strokeLinejoin="round"
          strokeWidth={strokeWidth}
          x1="16.0481"
          x2="15.341"
          y1="11.4519"
          y2="12.159"
        />
        <line
          fill="none"
          stroke="currentColor"
          strokeLinecap="round"
          strokeLinejoin="round"
          strokeWidth={strokeWidth}
          x1="17"
          x2="16"
          y1="13.75"
          y2="13.75"
        />
        <line
          fill="none"
          stroke="currentColor"
          strokeLinecap="round"
          strokeLinejoin="round"
          strokeWidth={strokeWidth}
          x1="16.0481"
          x2="15.341"
          y1="16.0481"
          y2="15.341"
        />
        <line
          fill="none"
          stroke="currentColor"
          strokeLinecap="round"
          strokeLinejoin="round"
          strokeWidth={strokeWidth}
          x1="13.75"
          x2="13.75"
          y1="17"
          y2="16"
        />
        <line
          fill="none"
          stroke="currentColor"
          strokeLinecap="round"
          strokeLinejoin="round"
          strokeWidth={strokeWidth}
          x1="11.4519"
          x2="12.159"
          y1="16.0481"
          y2="15.341"
        />
        <line
          fill="none"
          stroke="currentColor"
          strokeLinecap="round"
          strokeLinejoin="round"
          strokeWidth={strokeWidth}
          x1="10.5"
          x2="11.5"
          y1="13.75"
          y2="13.75"
        />
        <line
          fill="none"
          stroke="currentColor"
          strokeLinecap="round"
          strokeLinejoin="round"
          strokeWidth={strokeWidth}
          x1="11.4519"
          x2="12.159"
          y1="11.4519"
          y2="12.159"
        />
      </g>
    </svg>
  );
};
