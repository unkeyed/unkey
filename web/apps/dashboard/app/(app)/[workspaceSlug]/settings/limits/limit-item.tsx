"use client";

import { IconCircleInfoOutline12 } from "@unkey/icons";
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
import { cn } from "cn";
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
              <IconCircleInfoOutline12 className="shrink-0 text-gray-9" />
            </InfoTooltip>
          ) : null}
          {row.status === "ok" ? null : (
            <Badge variant="error" size="sm">
              {row.status === "over" ? "Over limit" : "At limit"}
            </Badge>
          )}
        </div>
      </ItemContent>
      <ItemActions className="w-56 sm:w-80">
        <LimitValue row={row} />
      </ItemActions>
    </Item>
  );
}

/** Content-width limit column, so the number stays flush right whatever its length. */
const CELLS = "grid w-full grid-cols-[5.5rem_1fr_auto] items-center gap-3";

function LimitValue({ row }: { row: LimitRow }) {
  const value = row.value;
  if (value.state === "loading") {
    return (
      <div className={CELLS}>
        {value.metered ? (
          <>
            <Skeleton className="h-4 w-14 justify-self-end" />
            <Skeleton className="h-1.5 w-full rounded-full" />
          </>
        ) : (
          <>
            <span />
            <span />
          </>
        )}
        <Skeleton className="h-4 w-16 justify-self-end" />
      </div>
    );
  }

  const usage = value.usage;
  if (!usage) {
    return (
      <div className={CELLS}>
        <span />
        <span />
        <Limit>{value.limit}</Limit>
      </div>
    );
  }

  const breached = row.status !== "ok";
  return (
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
      <Limit>{value.limit}</Limit>
    </Meter>
  );
}

function Limit({ children }: { children: ReactNode }) {
  return <span className="text-right tabular-nums">{children}</span>;
}
