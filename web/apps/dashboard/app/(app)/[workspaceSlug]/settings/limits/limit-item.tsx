"use client";

import { CircleInfo } from "@unkey/icons";
import {
  Badge,
  InfoTooltip,
  Item,
  ItemActions,
  ItemContent,
  ItemTitle,
  Meter,
  MeterHeader,
  MeterIndicator,
  MeterTrack,
  MeterValue,
  Skeleton,
} from "@unkey/ui";
import type { LimitRow } from "./limit-groups";

export function LimitItem({ row }: { row: LimitRow }) {
  return (
    <Item>
      <ItemContent>
        {/* The name line reserves the badge's height on every row, so crossing a
            ceiling adds a badge without moving the rows under it. */}
        <div className="flex h-5 items-center gap-2">
          <ItemTitle>{row.name}</ItemTitle>
          {row.description ? (
            <InfoTooltip content={row.description} position={{ side: "right" }}>
              <CircleInfo iconSize="sm-regular" className="shrink-0 text-gray-9" />
            </InfoTooltip>
          ) : null}
          {row.status === "ok" ? null : (
            <Badge variant="error" size="sm">
              {row.status === "over" ? "Over limit" : "At limit"}
            </Badge>
          )}
        </div>
      </ItemContent>
      <ItemActions className="w-64 flex-col items-stretch gap-1.5">
        <LimitValue row={row} />
      </ItemActions>
    </Item>
  );
}

function LimitValue({ row }: { row: LimitRow }) {
  const usage = row.usage;

  if (!usage) {
    return <span className="text-right tabular-nums">{row.limit}</span>;
  }

  if (usage.state === "loading") {
    return (
      <>
        <div className="flex items-baseline justify-between gap-3">
          <Skeleton className="h-3 w-14" />
          <span className="tabular-nums">{row.limit}</span>
        </div>
        <Skeleton className="h-1.5 w-full rounded-full" />
      </>
    );
  }

  if (usage.state === "error") {
    return (
      <div className="flex items-baseline justify-between gap-3">
        <span className="text-xs text-gray-9">Usage unavailable</span>
        <span className="tabular-nums">{row.limit}</span>
      </div>
    );
  }

  const breached = row.status !== "ok";

  return (
    <Meter aria-label={row.name} value={usage.value} max={usage.max} className="gap-1.5">
      <MeterHeader className="gap-3">
        <MeterValue className={breached ? "text-error-11" : "font-normal text-gray-11"}>
          {() => usage.label}
        </MeterValue>
        <span className="tabular-nums">{row.limit}</span>
      </MeterHeader>
      <MeterTrack>
        <MeterIndicator className={breached ? "bg-error-9" : undefined} />
      </MeterTrack>
    </Meter>
  );
}
