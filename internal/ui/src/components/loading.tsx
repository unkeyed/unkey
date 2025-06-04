// biome-ignore lint: React in this context is used throughout, so biome will change to types because no APIs are used even though React is needed.
import React from "react";
import type { SVGProps } from "react";
import { AnimatedLoadingSpinner } from "./animated-loading-spinner";
import { cn } from "../lib/utils";

interface LoadingProps extends SVGProps<SVGSVGElement> {
  /** Animation duration, e.g. "124ms" per segment  for spinner type */
  spinDurMS?: number;
  /** Animation duration, e.g. "0.75s" for dots type */
  dotsDurSeconds?: number;
  size?: number;
  type?: "spinner" | "dots";
}

export function Loading({
  size,
  spinDurMS,
  dotsDurSeconds,
  className,
  type = "spinner",
  ...props
}: LoadingProps): JSX.Element {
  if (type === "dots") {
    return <DotsLoading size={size ?? 24} className={className} dur={dotsDurSeconds} {...props} />;
  }
  return (
    <AnimatedLoadingSpinner
      size={size ?? 18}
      segmentTimeInMS={spinDurMS}
      className={cn("fill-current", className)}
      {...props}
    />
  );
}

type DotsLoadingProps = SVGProps<SVGSVGElement> & {
  size?: number;
  timeInSeconds?: number;
};

const DotsLoading = ({
  size,
  className,
  timeInSeconds,
  ...props
}: DotsLoadingProps): JSX.Element => {
  const dur = timeInSeconds ? `${timeInSeconds}s` : "0.75s";
  return (
    <svg
      {...props}
      className={cn("fill-current", className)}
      width={size}
      height={size}
      viewBox="0 0 24 24"
      xmlns="http://www.w3.org/2000/svg"
    >
      <circle cx="4" cy="12" r="3">
        <animate id="a" begin="0;b.end-0.25s" attributeName="r" dur={dur} values="3;.2;3" />
      </circle>
      <circle cx="12" cy="12" r="3">
        <animate begin="a.end-0.6s" attributeName="r" dur={dur} values="3;.2;3" />
      </circle>
      <circle cx="20" cy="12" r="3">
        <animate id="b" begin="a.end-0.45s" attributeName="r" dur={dur} values="3;.2;3" />
      </circle>
    </svg>
  );
};
