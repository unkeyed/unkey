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
import type { IconProps } from "../props";

export const TimeClock: React.FC<IconProps> = (props) => {
	return (
		<svg {...props} height="18" width="18" viewBox="0 0 18 18" xmlns="http://www.w3.org/2000/svg">
			<g fill="currentColor" strokeLinecap="butt" strokeLinejoin="miter">
				<line fill="none" stroke="currentColor" strokeLinecap="square" strokeMiterlimit="10" strokeWidth="1.5" x1="12" x2="12" y1="2.25" y2="4" />
				<line fill="none" stroke="currentColor" strokeLinecap="square" strokeMiterlimit="10" strokeWidth="1.5" x1="21.75" x2="20" y1="12" y2="12" />
				<line fill="none" stroke="currentColor" strokeLinecap="square" strokeMiterlimit="10" strokeWidth="1.5" x1="12" x2="12" y1="21.75" y2="20" />
				<line fill="none" stroke="currentColor" strokeLinecap="square" strokeMiterlimit="10" strokeWidth="1.5" x1="2.25" x2="4" y1="12" y2="12" />
				<circle cx="12" cy="12" fill="none" r="10" stroke="currentColor" strokeLinecap="square" strokeMiterlimit="10" strokeWidth="1.5" />
				<polyline fill="none" points="8 7 12 12 16 12" stroke="currentColor" strokeLinecap="square" strokeMiterlimit="10" strokeWidth="1.5" />
			</g>
		</svg>
	);
};

