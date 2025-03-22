// Clean external types that map to detailed internal types
type Size = "sm" | "md" | "lg" | "xl" | "2xl";
type Weight = "thin" | "regular" | "medium" | "bold";

type IconSize = `${Size}-${Weight}`;

export const sizeMap = {
  "sm-thin": { size: 12, strokeWidth: 1 },
  "sm-medium": { size: 12, strokeWidth: 1.5 },
  "sm-regular": { size: 12, strokeWidth: 2 },
  "sm-bold": { size: 12, strokeWidth: 3 },
  "md-thin": { size: 14, strokeWidth: 1 },
  "md-medium": { size: 14, strokeWidth: 1.5 },
  "md-regular": { size: 14, strokeWidth: 2 },
  "md-bold": { size: 14, strokeWidth: 3 },
  "lg-thin": { size: 16, strokeWidth: 1 },
  "lg-medium": { size: 16, strokeWidth: 1.5 },
  "lg-regular": { size: 16, strokeWidth: 2 },
  "lg-bold": { size: 16, strokeWidth: 3 },
  "xl-thin": { size: 18, strokeWidth: 1 },
  "xl-medium": { size: 18, strokeWidth: 1.5 },
  "xl-regular": { size: 18, strokeWidth: 2 },
  "xl-bold": { size: 18, strokeWidth: 3 },
  "2xl-thin": { size: 30, strokeWidth: 1 },
  "2xl-medium": { size: 30, strokeWidth: 1.5 },
  "2xl-regular": { size: 30, strokeWidth: 2 },
  "2xl-bold": { size: 30, strokeWidth: 3 },
} as const;

export type IconProps = {
  className?: string;
  title?: string;
  size?: IconSize;
  filled?: boolean;
};
