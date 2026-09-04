"use client";

import { CircleXMark } from "@unkey/icons";
import { Badge, Button, Card, CopyButton, Skeleton, TimestampInfo, cn } from "@unkey/ui";
import type { ReactNode } from "react";
import { type Delivery, detailText, isFailure, useDeliveries } from "./use-deliveries";

export function DrainLogsTable({ drainId }: { drainId: string }) {
  const { deliveries, isError, retry } = useDeliveries(drainId);

  return (
    <Card className="overflow-hidden">
      <div className="border-b border-gray-4 px-4 py-3">
        <span className="text-[13px] font-medium text-accent-12">Deliveries</span>
      </div>
      <div className="overflow-x-auto">
        <LogRows deliveries={deliveries} isError={isError} retry={retry} />
      </div>
    </Card>
  );
}

const SKELETON_ROWS = 5;

function LogRows({
  deliveries,
  isError,
  retry,
}: {
  deliveries: Delivery[] | undefined;
  isError: boolean;
  retry: () => void;
}) {
  if (isError) {
    return (
      <div className="flex flex-col items-center gap-3 px-4 py-8 text-center">
        <span role="alert" className="text-xs text-gray-11">
          We couldn't load delivery attempts.
        </span>
        <Button variant="outline" size="sm" onClick={retry}>
          Retry
        </Button>
      </div>
    );
  }
  if (!deliveries) {
    return (
      <div aria-busy="true" aria-live="polite" className="divide-y divide-gray-4">
        <output className="sr-only">Loading delivery attempts…</output>
        {Array.from({ length: SKELETON_ROWS }).map((_, index) => (
          <div
            // biome-ignore lint/suspicious/noArrayIndexKey: skeleton rows are static and never reorder
            key={index}
            aria-hidden="true"
            className="flex items-center gap-4 px-4 py-2.5"
          >
            <Skeleton className="h-3.5 w-32" />
            <Skeleton className="h-5 w-12 rounded-md" />
            <Skeleton className="h-3.5 w-8" />
            <Skeleton className="h-3.5 w-12" />
            <Skeleton className="h-3.5 flex-1" />
          </div>
        ))}
      </div>
    );
  }
  if (deliveries.length === 0) {
    return (
      <div className="px-4 py-8 text-center text-xs text-gray-9">No delivery attempts yet.</div>
    );
  }

  return (
    <table
      aria-label="Recent delivery attempts"
      className="w-full min-w-[680px] table-fixed border-collapse text-left"
    >
      <colgroup>
        <col className="w-40" />
        <col className="w-24" />
        <col className="w-20" />
        <col className="w-20" />
        <col />
      </colgroup>
      <thead>
        <tr className="border-b border-gray-4 bg-grayA-2">
          <Th>Time</Th>
          <Th>Status</Th>
          <Th>Events</Th>
          <Th>Latency</Th>
          <Th>Response</Th>
        </tr>
      </thead>
      <tbody className="divide-y divide-gray-4">
        {deliveries.map((delivery, index) => (
          <tr
            key={`${delivery.time}-${index}`}
            className="group transition-colors hover:bg-grayA-2"
          >
            <td className="px-4 py-2">
              <TimestampInfo
                value={delivery.time}
                className="whitespace-nowrap font-mono text-xs text-gray-11 decoration-dotted group-hover:underline"
              />
            </td>
            <td className="px-4 py-2">
              <StatusBadge delivery={delivery} />
            </td>
            <td className="px-4 py-2">
              <span className="font-mono text-xs tabular-nums text-gray-11">{delivery.events}</span>
            </td>
            <td className="px-4 py-2">
              <span className="font-mono text-xs tabular-nums text-gray-11">
                {delivery.durationMs}ms
              </span>
            </td>
            <td className="px-4 py-2">
              <ResponseCell delivery={delivery} />
            </td>
          </tr>
        ))}
      </tbody>
    </table>
  );
}

function Th({ children }: { children: ReactNode }) {
  return (
    <th scope="col" className="px-4 py-2 text-xs font-medium text-gray-10">
      {children}
    </th>
  );
}

function StatusBadge({ delivery }: { delivery: Delivery }) {
  const failed = isFailure(delivery);
  let label: string;
  if (delivery.responseStatus > 0) {
    label = String(delivery.responseStatus);
  } else if (failed) {
    label = "Error";
  } else {
    label = "OK";
  }

  return (
    <Badge
      className={cn(
        "whitespace-nowrap rounded-md px-[6px] font-mono uppercase",
        failed
          ? "border-transparent bg-error-4 text-error-11 group-hover:bg-error-5"
          : "border-transparent bg-grayA-3 text-grayA-11 group-hover:bg-grayA-5",
      )}
    >
      {label}
    </Badge>
  );
}

function ResponseCell({ delivery }: { delivery: Delivery }) {
  const text = detailText(delivery);
  if (text === "") {
    return <span className="font-mono text-xs text-gray-8">No response body</span>;
  }

  return (
    <div className="flex items-center gap-2">
      {isFailure(delivery) ? (
        <CircleXMark iconSize="sm-medium" className="size-3.5 shrink-0 text-error-11" />
      ) : null}
      <span className="min-w-0 flex-1 truncate font-mono text-xs text-gray-12">{text}</span>
      <CopyButton value={text} variant="ghost" size="sm" className="shrink-0" />
    </div>
  );
}
