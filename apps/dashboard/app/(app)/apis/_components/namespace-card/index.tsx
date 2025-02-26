"use client";
import { Clock, ProgressBar } from "@unkey/icons";
import ms from "ms";
import Link from "next/link";
import type { API } from "../../page";
import { useFetchVerificationTimeseries } from "../hooks/use-query-timeseries";
import { VerificationTimeseriesBarChart } from "./chart/bar-chart";

type Props = {
  api: API;
};

export const NamespaceCard = ({ api }: Props) => {
  const { timeseries, isLoading, isError } = useFetchVerificationTimeseries(api.keyspaceId);

  const passed = timeseries?.reduce((acc, crr) => acc + crr.success, 0) ?? 0;
  const blocked = timeseries?.reduce((acc, crr) => acc + crr.error, 0) ?? 0;

  const lastRequest = timeseries
    ? timeseries
        .filter((entry) => entry.total > 0)
        .sort((a, b) => b.originalTimestamp - a.originalTimestamp)[0]
    : null;

  return (
    <div className="flex flex-col border border-gray-6 rounded-xl overflow-hidden">
      <div className="h-[120px]">
        <VerificationTimeseriesBarChart
          data={timeseries}
          isLoading={isLoading}
          isError={isError}
          config={{
            success: {
              label: "Valid",
              color: "hsl(var(--accent-4))",
            },
            error: {
              label: "Invalid",
              color: "hsl(var(--orange-9))",
            },
          }}
        />
      </div>

      <Link href={`/apis/${api.id}`}>
        <div className="p-6 border-t border-gray-6 flex flex-col gap-2">
          <div className="flex justify-between items-center">
            <div className="flex flex-col">
              <div className="flex gap-3 items-center">
                <ProgressBar className="text-accent-11" />
                <div className="text-accent-12 font-semibold">{api.name}</div>
              </div>
              <div className="text-accent-11 text-xxs">{api.id}</div>
            </div>
            {api.keys.length !== undefined && (
              <div className="flex items-center justify-center rounded-full bg-accent-3 px-2 py-1">
                <span className="text-xs font-medium text-accent-11">{api.keys.length} API(s)</span>
              </div>
            )}
          </div>
          <div className="flex items-center w-full justify-between gap-4 mt-1">
            <div className="flex gap-[14px] items-center">
              <div className="flex flex-col gap-1">
                <div className="flex gap-2 items-center">
                  <div className="bg-accent-8 rounded h-[10px] w-1" />
                  <div className="text-accent-12 text-xs font-medium">{passed}</div>
                  <div className="text-accent-9 text-[11px] leading-4">VALID</div>
                </div>
              </div>
              <div className="flex flex-col gap-1">
                <div className="flex gap-2 items-center">
                  <div className="bg-orange-9 rounded h-[10px] w-1" />
                  <div className="text-accent-12 text-xs font-medium">{blocked}</div>
                  <div className="text-accent-9 text-[11px] leading-4">INVALID</div>
                </div>
              </div>
            </div>
            <div className="flex items-center gap-2 min-w-0 max-w-[40%]">
              <Clock className="text-accent-11 flex-shrink-0" />
              <div className="text-xs text-accent-9 truncate">
                {lastRequest
                  ? `${ms(Date.now() - lastRequest.originalTimestamp, {
                      long: true,
                    })} ago`
                  : "No data"}
              </div>
            </div>
          </div>
        </div>
      </Link>
    </div>
  );
};
