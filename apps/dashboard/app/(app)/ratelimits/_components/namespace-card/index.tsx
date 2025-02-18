"use client";
import { Clock, ProgressBar } from "@unkey/icons";
import ms from "ms";
import Link from "next/link";
import { useFetchRatelimitOverviewTimeseries } from "../../[namespaceId]/_overview/components/charts/bar-chart/hooks/use-fetch-timeseries";
import { LogsTimeseriesBarChart } from "./chart/bar-chart";

type Props = {
  namespace: {
    id: string;
    name: string;
  };
};

export const NamespaceCard = ({ namespace }: Props) => {
  const { timeseries } = useFetchRatelimitOverviewTimeseries(namespace.id);

  const passed = timeseries?.reduce((acc, crr) => acc + crr.success, 0) ?? 0;
  const blocked = timeseries?.reduce((acc, crr) => acc + crr.error, 0) ?? 0;

  const lastRatelimit = timeseries
    ? timeseries
        .filter((entry) => entry.total > 0)
        .sort((a, b) => b.originalTimestamp - a.originalTimestamp)[0]
    : null;

  return (
    <div className="flex flex-col border border-gray-6 rounded-xl overflow-hidden">
      <div className="h-[120px]">
        <LogsTimeseriesBarChart
          data={timeseries}
          config={{
            success: {
              label: "Passed",
              color: "hsl(var(--accent-4))",
            },
            error: {
              label: "Blocked",
              color: "hsl(var(--orange-9))",
            },
          }}
        />
      </div>
      <div className="p-6 border-t border-gray-6 flex flex-col gap-1">
        <div className="flex gap-3 items-center">
          <ProgressBar className="text-accent-11" />
          <Link className="text-accent-12 font-semibold" href={`/ratelimits/${namespace.id}`}>
            <div className="text-accent-12 font-semibold">{namespace.name}</div>
          </Link>
        </div>
        <div className="flex items-center w-full justify-between gap-10">
          <div className="flex gap-[14px] items-center">
            <div className="flex flex-col gap-1">
              <div className="flex gap-2 items-center">
                <div className="bg-accent-8 rounded h-[10px] w-1" />
                <div className="text-accent-12 text-xs font-medium">{passed}</div>
                <div className="text-accent-9 text-[11px] leading-4">PASSED</div>
              </div>
            </div>
            <div className="flex flex-col gap-1">
              <div className="flex gap-2 items-center">
                <div className="bg-orange-9 rounded h-[10px] w-1" />
                <div className="text-accent-12 text-xs font-medium">{blocked}</div>
                <div className="text-accent-9 text-[11px] leading-4">BLOCKED</div>
              </div>
            </div>
          </div>
          <div className="flex items-center gap-2">
            <Clock className="text-accent-11" />
            <div className="text-xs text-accent-9">
              {lastRatelimit
                ? `${ms(Date.now() - lastRatelimit.originalTimestamp, {
                    long: true,
                  })} ago`
                : "No data"}
            </div>
          </div>
        </div>
      </div>
    </div>
  );
};
