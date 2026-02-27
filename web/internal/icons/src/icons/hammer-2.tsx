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

export function Hammer2({ iconSize = "xl-thin", filled, ...props }: IconProps) {
  const { iconSize: pixelSize, strokeWidth } = sizeMap[iconSize];

  return (
    <svg height={pixelSize} width={pixelSize} viewBox="0 0 18 18" xmlns="http://www.w3.org/2000/svg" {...props}>
      <g fill="currentColor">
        <path d="M11.529,8.721l-6.563,6.563c-.621,.621-1.629,.621-2.25,0h0c-.621-.621-.621-1.629,0-2.25l6.563-6.563" fill="none" stroke="currentColor" strokeLinecap="round" strokeLinejoin="round" strokeWidth={strokeWidth} />
        <path d="M16.207,8.944l-1.521,1.521c-.391,.391-1.024,.391-1.414,0L5.742,2.934l.934-.934,4.191,.493c.223,.026,.431,.127,.59,.286l4.75,4.75c.391,.391,.391,1.024,0,1.414Z" fill="none" stroke="currentColor" strokeLinecap="round" strokeLinejoin="round" strokeWidth={strokeWidth} />
      </g>
    </svg>
  );
};

