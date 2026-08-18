"use client";

import { cn } from "@/lib/utils";
import { CircleInfo } from "@unkey/icons";
import { P, match } from "@unkey/match";
import {
  Badge,
  InfoTooltip,
  Item,
  ItemActions,
  ItemContent,
  ItemTitle,
  Meter,
  MeterIndicator,
  MeterTrack,
  MeterValue,
  Skeleton,
} from "@unkey/ui";
import type { ReactNode } from "react";
import type { LimitRow } from "./limit-groups";

export function LimitItem({ row }: { row: LimitRow }) {
  return (
    <Item>
      <ItemContent>
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
      <ItemActions className="w-80">
        <LimitValue row={row} />
      </ItemActions>
    </Item>
  );
}

/** Content-width limit column, so the number stays flush right whatever its length. */
const CELLS = "grid w-full grid-cols-[5.5rem_1fr_auto] items-center gap-3";

function LimitValue({ row }: { row: LimitRow }) {
  const breached = row.status !== "ok";

  return match(row.usage)
    .with(P.nullish, () => (
      <div className={CELLS}>
        <span />
        <span />
        <Limit>{row.limit}</Limit>
      </div>
    ))
    .with({ state: "loading" }, () => (
      <div className={CELLS}>
        <Skeleton className="h-3 w-14 justify-self-end" />
        <Skeleton className="h-1.5 w-full rounded-full" />
        <Limit>{row.limit}</Limit>
      </div>
    ))
    .with({ state: "error" }, () => (
      <div className={CELLS}>
        <span className="col-span-2 text-right text-xs text-gray-9">Usage unavailable</span>
        <Limit>{row.limit}</Limit>
      </div>
    ))
    .with({ state: "ready" }, (usage) => (
      <Meter
        layout="inline"
        className={CELLS}
        aria-label={row.name}
        value={usage.value}
        max={Math.max(usage.max, 1)}
      >
        <MeterValue
          className={cn("text-right", breached ? "text-error-11" : "font-normal text-gray-11")}
        >
          {() => usage.label}
        </MeterValue>
        <MeterTrack>
          <MeterIndicator className={breached ? "bg-error-9" : undefined} />
        </MeterTrack>
        <Limit>{row.limit}</Limit>
      </Meter>
    ))
    .exhaustive();
}

function Limit({ children }: { children: ReactNode }) {
  return <span className="text-right tabular-nums">{children}</span>;
}
