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

export const ArrowDottedRotateAnticlockwise: React.FC<IconProps> = ({
  size = "xl-thin",
  ...props
}) => {
  const { size: pixelSize } = sizeMap[size];

  return (
    <svg
      height={pixelSize}
      width={pixelSize}
      {...props}
      viewBox="0 0 18 18"
      xmlns="http://www.w3.org/2000/svg"
    >
      <g fill="currentColor">
        <path
          d="M9,1c-2.488,0-4.774,1.157-6.268,3.048l-.117-.846c-.058-.41-.442-.696-.846-.64-.411,.057-.697,.436-.641,.846l.408,2.945c.053,.375,.374,.647,.742,.647,.034,0,.069-.002,.104-.007l2.944-.407c.41-.057,.697-.436,.641-.846-.057-.411-.443-.694-.846-.641l-1.448,.2c1.199-1.727,3.168-2.8,5.326-2.8,3.397,0,6.245,2.651,6.483,6.037,.027,.395,.357,.697,.747,.697,.018,0,.036,0,.054-.002,.413-.029,.725-.388,.695-.801-.293-4.167-3.798-7.431-7.979-7.431Z"
          fill="currentColor"
        />
        <path
          d="M15.985,11.082c-.383-.159-.821,.023-.98,.406-.159,.383,.023,.821,.406,.98s.821-.023,.98-.406-.023-.821-.406-.98Z"
          fill="currentColor"
        />
        <path
          d="M11.487,15.005c-.383,.158-.564,.597-.406,.98,.159,.382,.597,.564,.98,.406s.564-.597,.406-.98-.597-.564-.98-.406Z"
          fill="currentColor"
        />
        <path
          d="M6.513,15.005c-.383-.159-.821,.023-.98,.406s.023,.821,.406,.98,.821-.023,.98-.406c.159-.383-.023-.822-.406-.98Z"
          fill="currentColor"
        />
        <path
          d="M2.015,11.082c-.383,.159-.564,.597-.406,.98s.597,.564,.98,.406,.564-.597,.406-.98c-.159-.383-.597-.564-.98-.406Z"
          fill="currentColor"
        />
        <circle cx="14.127" cy="14.126" fill="currentColor" r=".75" />
        <circle cx="9" cy="16.25" fill="currentColor" r=".75" />
        <circle cx="3.873" cy="14.126" fill="currentColor" r=".75" />
        <circle cx="1.75" cy="9" fill="currentColor" r=".75" />
      </g>
    </svg>
  );
};
