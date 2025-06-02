// biome-ignore lint: React in this context is used throughout, so biome will change to types because no APIs are used even though React is needed.
import React from "react";
import type { SVGProps } from "react";

interface LoadingProps extends SVGProps<SVGSVGElement> {
  /** Animation duration, e.g. "0.75s" */
  dur?: string;
}

export function Loading({ width = 24, height = 24, dur = "0.75" }: LoadingProps): JSX.Element {
  return (
    <svg
      className="fill-current"
      width={width}
      height={height}
      viewBox="0 0 24 24"
      xmlns="http://www.w3.org/2000/svg"
    >
      <circle cx="4" cy="12" r="3">
        <animate
          id="a"
          begin="0;b.end-0.25s"
          attributeName="r"
          dur={dur}
          values="3;.2;3"
          repeatCount="indefinite"
        />
      </circle>
      <circle cx="12" cy="12" r="3">
        <animate
          begin="a.end-0.6s"
          attributeName="r"
          dur={dur}
          values="3;.2;3"
          repeatCount="indefinite"
        />
      </circle>
      <circle cx="20" cy="12" r="3">
        <animate
          id="b"
          begin="a.end-0.45s"
          attributeName="r"
          dur={dur}
          values="3;.2;3"
          repeatCount="indefinite"
        />
      </circle>
    </svg>
  );
}
