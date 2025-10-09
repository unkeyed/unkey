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

export const ChevronDown: React.FC<IconProps> = ({
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
        <path
          d="M9,13.5c-.192,0-.384-.073-.53-.22L2.22,7.03c-.293-.293-.293-.768,0-1.061s.768-.293,1.061,0l5.72,5.72,5.72-5.72c.293-.293,.768-.293,1.061,0s.293,.768,0,1.061l-6.25,6.25c-.146,.146-.338,.22-.53,.22Z"
          strokeWidth={strokeWidth}
          fill="currentColor"
        />
      </g>
    </svg>
  );
};
