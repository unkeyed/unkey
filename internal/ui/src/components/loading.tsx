// biome-ignore lint: React in this context is used throughout, so biome will change to types because no APIs are used even though React is needed.
import React from "react";
import type { SVGProps } from "react";
import { AnimatedLoadingSpinner } from "./animated-loading-spinner";

interface LoadingProps extends SVGProps<SVGSVGElement> {
  /** Animation duration, e.g. "124ms" per segment */
  spinDur?: number;
  dotsDur?: string;
  size?: number;
  type?: "spinner" | "dots";
}

export function Loading({
  size,
  spinDur = 125,
  dotsDur = "0.75s",
  className,
  type = "spinner",
  ...props
}: LoadingProps): JSX.Element {
  if (type === "dots") {
    return <DotsLoading size={size ?? 24} className={className} {...props} />;
  }
  return (
    <AnimatedLoadingSpinner
      size={size ?? 18}
      segmentTimeInMS={spinDur}
      className={className}
      {...props}
    />
  );
}

type DotsLoadingProps = SVGProps<SVGSVGElement> & {
  size?: number;
};

const DotsLoading = ({ size, className, ...props }: DotsLoadingProps): JSX.Element => {
  const dur = "0.75s";
  return (
    <svg
      {...props}
      className="fill-current"
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
