import type React from "react";

// Clean external types that map to detailed internal types
export type Size = "sm" | "md" | "lg" | "xl" | "2xl";
export type Weight = "thin" | "regular" | "medium" | "bold";

export type iconSize = `${Size}-${Weight}`;

export const sizeMap = {
  "sm-thin": { iconSize: 12, strokeWidth: 1 },
  "sm-medium": { iconSize: 12, strokeWidth: 1.5 },
  "sm-regular": { iconSize: 12, strokeWidth: 2 },
  "sm-bold": { iconSize: 12, strokeWidth: 3 },
  "md-thin": { iconSize: 14, strokeWidth: 1 },
  "md-medium": { iconSize: 14, strokeWidth: 1.5 },
  "md-regular": { iconSize: 14, strokeWidth: 2 },
  "md-bold": { iconSize: 14, strokeWidth: 3 },
  "lg-thin": { iconSize: 16, strokeWidth: 1 },
  "lg-medium": { iconSize: 16, strokeWidth: 1.5 },
  "lg-regular": { iconSize: 16, strokeWidth: 2 },
  "lg-bold": { iconSize: 16, strokeWidth: 3 },
  "xl-thin": { iconSize: 18, strokeWidth: 1 },
  "xl-medium": { iconSize: 18, strokeWidth: 1.5 },
  "xl-regular": { iconSize: 18, strokeWidth: 2 },
  "xl-bold": { iconSize: 18, strokeWidth: 3 },
  "2xl-thin": { iconSize: 30, strokeWidth: 1 },
  "2xl-medium": { iconSize: 30, strokeWidth: 1.5 },
  "2xl-regular": { iconSize: 30, strokeWidth: 2 },
  "2xl-bold": { iconSize: 30, strokeWidth: 3 },
} as const;

export type IconProps = {
  className?: string;
  title?: string;
  iconSize?: iconSize;
  filled?: boolean;
  focusable?: boolean;
  style?: React.CSSProperties;
};
