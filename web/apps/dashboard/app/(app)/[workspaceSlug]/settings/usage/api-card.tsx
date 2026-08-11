"use client";

import { formatDollars, formatNumber } from "@/lib/fmt";
import { Gauge, Key2, Nodes } from "@unkey/icons";
import {
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
import { Fragment, type ReactNode } from "react";

const BAND =
  "flex items-center gap-3 border-grayA-4 border-b bg-grayA-2 px-4 py-2 font-semibold text-[10px] text-gray-9 uppercase tracking-wider";

const AMOUNT = "font-semibold text-2xl text-gray-12 leading-tight tracking-tight tabular-nums";

const ROW_ICON = "size-5 border border-grayA-4 bg-gray-1";

const REQUESTS = "w-28";

type ApiCardProps = {
  verifications: number | null;
  ratelimits: number | null;
  /** Requests the plan includes per month, the ceiling for the quota bar. */
  quota: number | null;
  /** The flat monthly plan fee. There is no per-request rate to attribute. */
  feeCents: number | null;
  isLoading: boolean;
};

/**
 * Billing keeps no keyspace or namespace grain for either operation, so the two
 * workspace counts are the whole table.
 */
export function ApiCard({ verifications, ratelimits, quota, feeCents, isLoading }: ApiCardProps) {
  const rows: Array<{ icon: ReactNode; title: string; value: number | null }> = [
    { icon: <Key2 />, title: "Key verifications", value: verifications },
    { icon: <Gauge />, title: "Rate limit operations", value: ratelimits },
  ];
  const used = verifications === null || ratelimits === null ? null : verifications + ratelimits;

  return (
    <ItemGroup variant="outline">
      <ItemHeader>
        <ItemMedia className="bg-infoA-3 text-info-11">
          <Nodes />
        </ItemMedia>
        <ItemContent>
          <ItemTitle>API management</ItemTitle>
          <ItemDescription>Valid requests this period</ItemDescription>
        </ItemContent>
        <ItemActions className={AMOUNT}>
          <Figure
            value={feeCents === null ? null : formatDollars(feeCents)}
            isLoading={isLoading}
            skeletonClassName="h-6 w-16"
          />
        </ItemActions>
      </ItemHeader>

      <div className="px-4 pb-4">
        <Quota used={used} quota={quota} />
      </div>

      <div className={BAND}>
        <div className="min-w-0 flex-1">Operation</div>
        <div className={`${REQUESTS} text-right`}>Requests</div>
      </div>
      {rows.map((row, index) => (
        <Fragment key={row.title}>
          {index === 0 ? null : <ItemSeparator />}
          <Item>
            <ItemMedia className={ROW_ICON}>{row.icon}</ItemMedia>
            <ItemContent>
              <ItemTitle className="truncate">{row.title}</ItemTitle>
            </ItemContent>
            <ItemActions className={`${REQUESTS} justify-end tabular-nums`}>
              <Figure
                value={row.value === null ? null : row.value.toLocaleString("en-US")}
                isLoading={isLoading}
              />
            </ItemActions>
          </Item>
        </Fragment>
      ))}
    </ItemGroup>
  );
}

/**
 * The bar stays neutral at every level: exceeding the included quota is not an
 * error state here. Nothing is charged for it and nothing is blocked.
 */
function Quota({ used, quota }: { used: number | null; quota: number | null }) {
  if (used === null || quota === null) {
    return (
      <div className="flex flex-col gap-2">
        <Skeleton className="h-4 w-44" />
        <Skeleton className="h-1.5 w-full rounded-full" />
      </div>
    );
  }

  const percent = quota === 0 ? 0 : Math.round((used / quota) * 100);

  return (
    <Meter value={used} max={Math.max(quota, 1)}>
      <MeterHeader>
        <MeterLabel className="text-gray-12">
          {formatNumber(used)} of {formatNumber(quota)} requests
        </MeterLabel>
        <MeterValue className="font-normal text-gray-10 text-xs">{() => `${percent}%`}</MeterValue>
      </MeterHeader>
      <MeterTrack>
        <MeterIndicator />
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
  return <span className="font-normal text-[13px] text-gray-9">Unavailable</span>;
}
