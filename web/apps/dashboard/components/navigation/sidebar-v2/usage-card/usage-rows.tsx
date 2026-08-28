"use client";

import { formatCompactQuantity, formatDollars } from "@/lib/fmt";
import { cn } from "@/lib/utils";
import { Skeleton } from "@unkey/ui";
import type { ReactNode } from "react";
import {
  AT_RISK,
  type ApiUsage,
  type ComputeUsage,
  type Measured,
  apiRatio,
  computeRatio,
} from "./use-usage-summary";

export function ComputeRow({ measured }: { measured: Measured<ComputeUsage> }) {
  return (
    <Row
      label="Compute"
      measured={measured}
      ratioOf={computeRatio}
      renderValue={(value, ratio) =>
        value.budgetCents === null || value.budgetCents <= 0 ? (
          <span className="text-gray-12">{formatDollars(value.grossCents)}</span>
        ) : (
          <Fraction
            used={formatDollars(value.grossCents)}
            ceiling={formatDollars(value.budgetCents)}
            ratio={ratio}
          />
        )
      }
    />
  );
}

export function ApiRow({ measured }: { measured: Measured<ApiUsage> }) {
  return (
    <Row
      label="API"
      measured={measured}
      ratioOf={apiRatio}
      renderValue={(value, ratio) => (
        <Fraction
          used={formatCompactQuantity(value.used)}
          ceiling={formatCompactQuantity(value.max)}
          ratio={ratio}
        />
      )}
    />
  );
}

function Row<T>({
  label,
  measured,
  ratioOf,
  renderValue,
}: {
  label: string;
  measured: Measured<T>;
  ratioOf: (value: T) => number | null;
  renderValue: (value: T, ratio: number | null) => ReactNode;
}) {
  switch (measured.state) {
    case "loading":
      return (
        <Shell label={label} atRisk={false} value={<Skeleton className="h-3 w-16" />}>
          <Skeleton className="h-1 w-full rounded-full" />
        </Shell>
      );
    case "error":
      return <Shell label={label} atRisk={false} value={<span className="text-gray-8">—</span>} />;
    case "ready": {
      const ratio = ratioOf(measured.value);
      const atRisk = ratio !== null && ratio >= AT_RISK;
      return (
        <Shell label={label} atRisk={atRisk} value={renderValue(measured.value, ratio)}>
          {ratio === null ? null : <Bar ratio={ratio} atRisk={atRisk} />}
        </Shell>
      );
    }
    default:
      return measured satisfies never;
  }
}

function Shell({
  label,
  atRisk,
  value,
  children,
}: {
  label: string;
  atRisk: boolean;
  value: ReactNode;
  children?: ReactNode;
}) {
  return (
    <div className="flex flex-col gap-1.5">
      <div className="flex items-baseline gap-2">
        <span
          className={cn("flex-1 truncate text-[11px]", atRisk ? "text-error-9" : "text-gray-9")}
        >
          {label}
        </span>
        <span className="whitespace-nowrap text-[11px] tabular-nums">{value}</span>
      </div>
      {children}
    </div>
  );
}

function Fraction({
  used,
  ceiling,
  ratio,
}: {
  used: string;
  ceiling: string;
  ratio: number | null;
}) {
  if (ratio !== null && ratio >= AT_RISK) {
    return (
      <span className="text-error-9">
        {used} / {ceiling}
      </span>
    );
  }
  return (
    <>
      <span className="text-gray-12">{used}</span>
      <span className="text-gray-7"> / </span>
      <span className="text-gray-12">{ceiling}</span>
    </>
  );
}

function Bar({ ratio, atRisk }: { ratio: number; atRisk: boolean }) {
  return (
    <div className="h-1 w-full overflow-hidden rounded-full bg-grayA-4">
      <div
        className={cn("h-full rounded-full", atRisk ? "bg-error-9" : "bg-gray-12")}
        style={{ width: `${Math.min(Math.max(ratio, 0), 1) * 100}%` }}
      />
    </div>
  );
}
