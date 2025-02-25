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

export const CaretDown: React.FC<IconProps> = ({ size = "xl-thin", ...props }) => {
  const { size: pixelSize, strokeWidth } = sizeMap[size];
  return (
    <svg
      {...props}
      height={pixelSize}
      width={pixelSize}
      viewBox="0 0 18 18"
      xmlns="http://www.w3.org/2000/svg"
    >
      <g fill="currentColor">
        <path
          d="M9.845,14.209l5.025-7.923c.422-.666-.056-1.536-.845-1.536H3.975c-.788,0-1.267,.87-.845,1.536l5.025,7.923c.393,.619,1.296,.619,1.689,0Z"
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

<svg height="18" width="18" viewBox="0 0 18 18" xmlns="http://www.w3.org/2000/svg">
  <g fill="#212121">
    <path
      d="M14.024,4H3.976c-.638,0-1.226,.347-1.533,.906s-.287,1.242,.055,1.781l5.024,7.923c.323,.509,.875,.812,1.478,.812s1.155-.304,1.478-.812l5.025-7.924c.341-.539,.362-1.222,.055-1.781s-.895-.906-1.533-.906Z"
      fill="#212121"
    />
  </g>
</svg>;
