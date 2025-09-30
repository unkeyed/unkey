import React from "react";

// Clean external types that map to detailed internal types
export type Size = "sm" | "md" | "lg" | "xl" | "2xl";
export type Weight = "thin" | "regular" | "medium" | "bold";

export type iconsize = `${Size}-${Weight}`;

export const sizeMap = {
  "sm-thin": { iconsize: 12, strokeWidth: 1 },
  "sm-medium": { iconsize: 12, strokeWidth: 1.5 },
  "sm-regular": { iconsize: 12, strokeWidth: 2 },
  "sm-bold": { iconsize: 12, strokeWidth: 3 },
  "md-thin": { iconsize: 14, strokeWidth: 1 },
  "md-medium": { iconsize: 14, strokeWidth: 1.5 },
  "md-regular": { iconsize: 14, strokeWidth: 2 },
  "md-bold": { iconsize: 14, strokeWidth: 3 },
  "lg-thin": { iconsize: 16, strokeWidth: 1 },
  "lg-medium": { iconsize: 16, strokeWidth: 1.5 },
  "lg-regular": { iconsize: 16, strokeWidth: 2 },
  "lg-bold": { iconsize: 16, strokeWidth: 3 },
  "xl-thin": { iconsize: 18, strokeWidth: 1 },
  "xl-medium": { iconsize: 18, strokeWidth: 1.5 },
  "xl-regular": { iconsize: 18, strokeWidth: 2 },
  "xl-bold": { iconsize: 18, strokeWidth: 3 },
  "2xl-thin": { iconsize: 30, strokeWidth: 1 },
  "2xl-medium": { iconsize: 30, strokeWidth: 1.5 },
  "2xl-regular": { iconsize: 30, strokeWidth: 2 },
  "2xl-bold": { iconsize: 30, strokeWidth: 3 },
} as const;

export type IconProps = {
  className?: string;
  title?: string;
  iconsize?: iconsize;
  filled?: boolean;
  focusable?: boolean;
  style?: React.CSSProperties;
};
