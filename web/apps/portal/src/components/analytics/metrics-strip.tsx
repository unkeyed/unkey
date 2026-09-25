import type { VerificationMetrics } from "./analytics-transform";
import type { ChartState } from "./chart-slot";
import { formatCount } from "./format";
import { INVALID_COLOR, VALID_COLOR } from "./outcomes";

type Props = {
  metrics: VerificationMetrics;
  state: ChartState;
};

export function MetricsStrip({ metrics, state }: Props) {
  const stats = [
    { label: "Total requests", value: metrics.totalRequests },
    { label: "Valid", value: metrics.validRequests, swatch: VALID_COLOR },
    { label: "Invalid", value: metrics.errorRequests, swatch: INVALID_COLOR },
  ];

  return (
    <dl className="grid grid-cols-3 gap-4 px-5 py-5 sm:grid-cols-[repeat(3,140px)] sm:gap-10">
      {stats.map(({ label, value, swatch }, index) => (
        <div key={label} className="relative min-w-0">
          {index > 0 && (
            <span
              aria-hidden="true"
              className="-left-2 sm:-left-5 absolute inset-y-0 border-primary/10 border-l"
            />
          )}
          <dt className="flex items-center gap-2 text-gray-11 text-xs">
            {swatch && (
              <span
                className="size-2 shrink-0 rounded-xs"
                style={{ backgroundColor: swatch }}
                aria-hidden="true"
              />
            )}
            {label}
          </dt>
          <dd className="mt-1 font-semibold text-gray-12 text-lg tabular-nums">
            {state.kind === "loading" ? (
              <span
                className="mt-1 block h-5 w-20 rounded bg-gray-3 motion-safe:animate-pulse"
                aria-busy="true"
              />
            ) : state.kind === "error" ? (
              "--"
            ) : (
              formatCount(value)
            )}
          </dd>
        </div>
      ))}
    </dl>
  );
}
