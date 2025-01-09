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
import React from "react";
import type { IconProps } from "../props";

export const Conversion: React.FC<IconProps> = (props) => {
  return (
    <svg
      {...props}
      height="18"
      width="18"
      viewBox="0 0 18 18"
      xmlns="http://www.w3.org/2000/svg"
    >
      <g fill="currentColor">
        <path
          d="M2.75,8H15.25c.304,0,.577-.183,.693-.463,.115-.28,.052-.603-.163-.817L11.28,2.22c-.293-.293-.768-.293-1.061,0s-.293,.768,0,1.061l3.22,3.22H2.75c-.414,0-.75,.336-.75,.75s.336,.75,.75,.75Z"
          fill="currentColor"
        />
        <path
          d="M15.25,10H2.75c-.304,0-.577,.183-.693,.463-.115,.28-.052,.603,.163,.817l4.5,4.5c.146,.146,.338,.22,.53,.22s.384-.073,.53-.22c.293-.293,.293-.768,0-1.061l-3.22-3.22H15.25c.414,0,.75-.336,.75-.75s-.336-.75-.75-.75Z"
          fill="currentColor"
        />
      </g>
    </svg>
  );
};
