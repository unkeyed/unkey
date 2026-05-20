import type { CSSProperties } from "react";

// Clean external types that map to detailed internal types
export type Size = "sm" | "md" | "lg" | "xl" | "2xl";
export type Weight = "medium" | "regular";

export type iconSize = `${Size}-${Weight}`;

export const sizeMap = {
  "sm-medium": { iconSize: 12, strokeWidth: 1.5 },
  "sm-regular": { iconSize: 12, strokeWidth: 2 },
  "md-medium": { iconSize: 14, strokeWidth: 1.5 },
  "md-regular": { iconSize: 14, strokeWidth: 2 },
  "lg-medium": { iconSize: 16, strokeWidth: 1.5 },
  "lg-regular": { iconSize: 16, strokeWidth: 2 },
  "xl-medium": { iconSize: 18, strokeWidth: 1.5 },
  "xl-regular": { iconSize: 18, strokeWidth: 2 },
  "2xl-medium": { iconSize: 30, strokeWidth: 1.5 },
  "2xl-regular": { iconSize: 30, strokeWidth: 2 },
} as const;

// export type IconProps = {
//   className?: string;
//   title?: string;
//   iconSize?: iconSize;
//   filled?: boolean;
//   focusable?: boolean;
//   style?: CSSProperties;
// };

export interface IconProps {
  className?: string;
  title?: string;
  iconSize?: iconSize;
  filled?: boolean;
  focusable?: boolean;
  style?: CSSProperties;
}
