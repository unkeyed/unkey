// biome-ignore lint: React in this context is used throughout, so biome will change to types because no APIs are used even though React is needed.
import React from "react";
import type { SVGProps } from "react";
import { AnimatedLoadingSpinner } from "./animated-loading-spinner";

interface LoadingProps extends SVGProps<SVGSVGElement> {
  /** Animation duration, e.g. "124ms" per segment */
  dur?: number;
  size?: number;
}

export function Loading({ size = 24, dur = 125, className, ...props }: LoadingProps): JSX.Element {
  return (
    <AnimatedLoadingSpinner size={size} segmentTimeInMS={dur} className={className} {...props} />
  );
}
