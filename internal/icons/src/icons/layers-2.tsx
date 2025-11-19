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

export const Layers2: React.FC<IconProps> = ({
  iconSize = "xl-regular",
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
        <path
          d="M2.58,6.149L8.385,1.949c.367-.266,.864-.266,1.231,0l5.805,4.2c.579,.419,.579,1.282,0,1.701l-5.805,4.2c-.367,.266-.864,.266-1.231,0L2.58,7.851c-.579-.419-.579-1.282,0-1.701Z"
          fill="none"
          stroke="currentColor"
          strokeLinecap="round"
          strokeLinejoin="round"
          strokeWidth={strokeWidth}
        />
        <path
          d="M15.746,10.533c.217,.439,.109,1.003-.326,1.317l-5.805,4.2c-.184,.133-.4,.199-.615,.199-.216,0-.432-.066-.615-.199L2.58,11.851c-.434-.314-.543-.878-.326-1.317"
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

export default Layers2;
