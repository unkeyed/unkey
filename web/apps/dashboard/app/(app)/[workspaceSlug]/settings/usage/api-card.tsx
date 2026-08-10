"use client";

import { formatNumber } from "@/lib/fmt";
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
  Skeleton,
} from "@unkey/ui";
import { Fragment, type ReactNode } from "react";

type ApiCardProps = {
  verifications: number | null;
  ratelimits: number | null;
  isLoading: boolean;
};

/**
 * Billing keeps no keyspace or namespace grain for either operation, so the two
 * workspace counts are the whole table. The plan fee is a flat charge with no
 * per-request rate, so no row carries a cost.
 */
export function ApiCard({ verifications, ratelimits, isLoading }: ApiCardProps) {
  const rows: Array<{ icon: ReactNode; title: string; description: string; value: number | null }> =
    [
      {
        icon: <Key2 />,
        title: "Key verifications",
        description: "Valid verifications, excluding gateway traffic.",
        value: verifications,
      },
      {
        icon: <Gauge />,
        title: "Rate limit operations",
        description: "Rate limit requests that passed.",
        value: ratelimits,
      },
    ];

  return (
    <ItemGroup variant="outline">
      <ItemHeader>
        <ItemMedia className="bg-infoA-3 text-info-11">
          <Nodes />
        </ItemMedia>
        <ItemContent>
          <ItemTitle>API management</ItemTitle>
          <ItemDescription>Billable operations this period</ItemDescription>
        </ItemContent>
      </ItemHeader>
      {rows.map((row) => (
        <Fragment key={row.title}>
          <ItemSeparator />
          <Item>
            <ItemMedia>{row.icon}</ItemMedia>
            <ItemContent>
              <ItemTitle>{row.title}</ItemTitle>
              <ItemDescription>{row.description}</ItemDescription>
            </ItemContent>
            <ItemActions className="tabular-nums">
              {isLoading ? (
                <Skeleton className="h-3 w-14" />
              ) : row.value === null ? (
                <span className="text-gray-9">Unavailable</span>
              ) : (
                formatNumber(row.value)
              )}
            </ItemActions>
          </Item>
        </Fragment>
      ))}
    </ItemGroup>
  );
}
