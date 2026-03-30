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

export function Note3({ iconSize = "xl-thin", ...props }: IconProps) {
  const { iconSize: pixelSize, strokeWidth } = sizeMap[iconSize];

  return (
    <svg height={pixelSize} width={pixelSize} viewBox="0 0 18 18" xmlns="http://www.w3.org/2000/svg" {...props}>
      <g fill="currentColor">
        <line fill="none" stroke="currentColor" strokeLinecap="round" strokeLinejoin="round" strokeWidth={strokeWidth} x1="7.25" x2="5.75" y1="11.75" y2="11.75" />
        <line fill="none" stroke="currentColor" strokeLinecap="round" strokeLinejoin="round" strokeWidth={strokeWidth} x1="8" x2="5.75" y1="9" y2="9" />
        <line fill="none" stroke="currentColor" strokeLinecap="round" strokeLinejoin="round" strokeWidth={strokeWidth} x1="12.25" x2="5.75" y1="6.25" y2="6.25" />
        <path d="m4.75,2.75h8.5c1.1046,0,2,.8954,2,2v5.1716c0,.5304-.2107,1.0391-.5858,1.4142l-3.3284,3.3284c-.3751.3751-.8838.5858-1.4142.5858h-5.1716c-1.1046,0-2-.8954-2-2V4.75c0-1.1046.8954-2,2-2Z" fill="none" stroke="currentColor" strokeLinecap="round" strokeLinejoin="round" strokeWidth={strokeWidth} />
        <path d="m10.25,15.148v-3.898c0-.552.448-1,1-1h3.91" fill="none" stroke="currentColor" strokeLinecap="round" strokeLinejoin="round" strokeWidth={strokeWidth} />
      </g>
    </svg>
  );
};
