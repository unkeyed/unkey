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

export const BookOpen: React.FC<IconProps> = ({
  size = "xl-thin",
  ...props
}) => {
  const { size: pixelSize, strokeWidth } = sizeMap[size];
  return (
    <svg
      height={pixelSize}
      width={pixelSize}
      {...props}
      viewBox="0 0 24 24"
      xmlns="http://www.w3.org/2000/svg"
    >
      <g
        fill="none"
        strokeLinecap="round"
        strokeLinejoin="round"
        transform="translate(0.25 0.25)"
      >
        <path
          d="M12 20.25V20.5V4.75V5"
          fill="none"
          stroke="currentColor"
          strokeWidth={strokeWidth}
        />
        <path
          d="M12 20.4999C15.5002 18.4997 19.4998 18.4997 23 20.4999V4.99993C19.4998 2.9997 15.5002 2.9997 12 4.99993C8.49975 2.9997 4.50025 2.99987 1 5.0001V20.4999C4.50025 18.4997 8.49975 18.4997 12 20.4999Z"
          fill="none"
          stroke="currentColor"
          strokeWidth={strokeWidth}
        />
      </g>
    </svg>
  );
};
