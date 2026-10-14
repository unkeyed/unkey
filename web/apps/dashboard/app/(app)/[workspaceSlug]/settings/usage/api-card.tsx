"use client";

import { AT_RISK } from "@/components/navigation/sidebar-v2/usage-card/use-usage-summary";
import { formatDollars, formatNumber, formatPrice } from "@/lib/fmt";
import { IconGaugeOutline18, IconKey2Outline18, IconNodesOutline18 } from "@unkey/icons";
import {
  Button,
  Item,
  ItemActions,
  ItemContent,
  ItemDescription,
  ItemGroup,
  ItemHeader,
  ItemMedia,
  ItemSeparator,
  ItemTitle,
  Meter,
  MeterHeader,
  MeterIndicator,
  MeterLabel,
  MeterTrack,
  MeterValue,
  Skeleton,
} from "@unkey/ui";
import { cn } from "cn";
import { Fragment, type ReactNode, useState } from "react";
import { PlansScreen } from "../billing/components/plans-screen";
import type { UsagePeriodRange } from "./period";

type ApiCardProps = {
  verifications: number | null;
  ratelimits: number | null;
  quota: number | null;
  feeCents: number | null;
  range: UsagePeriodRange;
  isLoading: boolean;
  isError: boolean;
  canUpgrade: boolean;
};

type MeterTone = "normal" | "atRisk";

const METER_TONES: Record<MeterTone, { label: string; value: string; indicator?: string }> = {
  normal: { label: "text-gray-12", value: "text-gray-10" },
  atRisk: { label: "text-error-11", value: "text-error-11", indicator: "bg-error-9" },
};

function days(count: number): string {
  return count === 1 ? "1 day" : `${count} days`;
}

function fee(feeCents: number | null): string | null {
  if (feeCents === null) {
    return null;
  }
  return feeCents === 0 ? formatPrice(0) : formatDollars(feeCents);
}

export function ApiCard({
  verifications,
  ratelimits,
  quota,
  feeCents,
  range,
  isLoading,
  isError,
  canUpgrade,
}: ApiCardProps) {
  const [plansOpen, setPlansOpen] = useState(false);
  const rows: Array<{ icon: ReactNode; title: string; value: number | null }> = [
    { icon: <IconKey2Outline18 />, title: "Key verifications", value: verifications },
    { icon: <IconGaugeOutline18 />, title: "Rate limit operations", value: ratelimits },
  ];
  const used = verifications === null || ratelimits === null ? null : verifications + ratelimits;
  const nearLimit = used !== null && quota !== null && usageRatio(used, quota) >= AT_RISK;

  return (
    <ItemGroup variant="outline">
      <ItemHeader>
        <ItemMedia className="bg-infoA-3 text-info-11">
          <IconNodesOutline18 />
        </ItemMedia>
        <ItemContent>
          <ItemTitle>API management</ItemTitle>
          <ItemDescription>
            Valid requests, {range.label}
            {range.daysLeft === null ? null : ` · resets in ${days(range.daysLeft)}`}
          </ItemDescription>
        </ItemContent>
        {canUpgrade ? (
          <ItemActions>
            <Button
              variant={nearLimit ? "primary" : "outline"}
              size="sm"
              aria-label="Upgrade API plan"
              onClick={() => setPlansOpen(true)}
            >
              Upgrade
            </Button>
          </ItemActions>
        ) : null}
        <ItemActions className="font-semibold text-2xl text-gray-12 leading-tight tracking-tight tabular-nums">
          <Figure value={fee(feeCents)} isLoading={isLoading} skeletonClassName="h-6 w-16" />
        </ItemActions>
      </ItemHeader>

      <div className="px-4 pb-4">
        <Quota used={used} quota={quota} isError={isError} />
      </div>

      <div className="flex items-center gap-3 border-b bg-gray-2 px-4 py-2 font-semibold text-3xs text-gray-9 uppercase tracking-wider">
        <div className="min-w-0 flex-1">Operation</div>
        <div className="w-28 text-right">Requests</div>
      </div>
      {rows.map((row, index) => (
        <Fragment key={row.title}>
          {index === 0 ? null : <ItemSeparator />}
          <Item>
            <ItemMedia className="size-5 border bg-raised">{row.icon}</ItemMedia>
            <ItemContent>
              <ItemTitle className="truncate">{row.title}</ItemTitle>
            </ItemContent>
            <ItemActions className="w-28 justify-end tabular-nums">
              <Figure
                value={row.value === null ? null : row.value.toLocaleString("en-US")}
                isLoading={isLoading}
              />
            </ItemActions>
          </Item>
        </Fragment>
      ))}
      <PlansScreen open={plansOpen} onOpenChange={setPlansOpen} reason="api-limit" />
    </ItemGroup>
  );
}

function usageRatio(used: number, quota: number): number {
  if (quota === 0) {
    return used === 0 ? 0 : 1;
  }
  return used / quota;
}

function Quota({
  used,
  quota,
  isError,
}: {
  used: number | null;
  quota: number | null;
  isError: boolean;
}) {
  if (isError) {
    return (
      <div className="flex flex-col gap-2">
        <div className="flex h-5 items-center">
          <span className="text-sm text-gray-9">Usage unavailable</span>
        </div>
        <div className="h-1.5 w-full rounded-full bg-grayA-3" />
      </div>
    );
  }

  if (used === null || quota === null) {
    return (
      <div className="flex flex-col gap-2">
        <div className="flex h-5 items-center justify-between gap-4">
          <Skeleton className="h-4 w-44" />
          <Skeleton className="h-3 w-8" />
        </div>
        <Skeleton className="h-1.5 w-full rounded-full" />
      </div>
    );
  }

  const ratio = usageRatio(used, quota);
  const percent = Math.round(ratio * 100);
  const tone = METER_TONES[ratio >= AT_RISK ? "atRisk" : "normal"];

  return (
    <Meter value={used} max={Math.max(quota, 1)}>
      <MeterHeader>
        <MeterLabel className={tone.label}>
          {formatNumber(used)} of {formatNumber(quota)} requests
        </MeterLabel>
        <MeterValue className={cn("font-normal text-xs", tone.value)}>
          {() => `${percent}%`}
        </MeterValue>
      </MeterHeader>
      <MeterTrack>
        <MeterIndicator className={tone.indicator} />
      </MeterTrack>
    </Meter>
  );
}

function Figure({
  value,
  isLoading,
  skeletonClassName = "h-3 w-14",
}: {
  value: string | null;
  isLoading: boolean;
  skeletonClassName?: string;
}) {
  if (value !== null) {
    return <>{value}</>;
  }
  if (isLoading) {
    return <Skeleton className={skeletonClassName} />;
  }
  return <span className="font-normal text-sm text-gray-9">Unavailable</span>;
}
