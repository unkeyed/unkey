"use client";

import { formatNumber } from "@/lib/fmt";
import { Button, Card } from "@unkey/ui";
import { WINDOW_HOURS, useDeliveryTotals } from "./use-deliveries";

const NO_VALUE = "‒";

export function DrainStatsCards({ drainId }: { drainId: string }) {
  const { totals, isError, retry } = useDeliveryTotals(drainId);

  return (
    <div className="flex flex-col gap-2">
      <div className="flex items-baseline justify-between gap-2">
        <span className="text-[13px] font-medium text-accent-12">Deliveries</span>
        {isError ? (
          <span role="alert" className="flex items-center gap-2 text-xs text-gray-11">
            We couldn't load delivery metrics.
            <Button variant="ghost" size="sm" onClick={retry}>
              Retry
            </Button>
          </span>
        ) : (
          <span className="text-xs text-gray-9">Past {WINDOW_HOURS} hours</span>
        )}
      </div>

      <div className="grid grid-cols-1 gap-4 md:grid-cols-3">
        <StatCard label="Delivered" value={totals ? formatNumber(totals.delivered) : NO_VALUE} />
        <StatCard label="Failed" value={totals ? formatNumber(totals.failed) : NO_VALUE} />
        <StatCard
          label="Success rate"
          value={
            totals === null || totals.successRate === null
              ? NO_VALUE
              : `${totals.successRate.toFixed(1)}%`
          }
        />
      </div>
    </div>
  );
}

function StatCard({ label, value }: { label: string; value: string }) {
  return (
    <Card>
      <div className="flex flex-col p-4">
        <span className="text-[13px] leading-5 text-content-subtle">{label}</span>
        <span className="text-[22px] font-semibold leading-tight tracking-tight tabular-nums">
          {value}
        </span>
      </div>
    </Card>
  );
}
