"use client";

import { cn } from "@/lib/utils";
import type { Route } from "next";
import Link from "next/link";
import type { ReactNode } from "react";
import {
  CountPill,
  IdentitiesRow,
  ROW_HEIGHT,
  RowLink,
  RowSkeleton,
  SectionLabel,
  fmt,
} from "../parts";
import type { LaunchpadRow, VariantProps } from "../types";
import { DeployNudge, GetStarted, RowMeta, RowSpark, Value } from "./shared";

function Card({
  title,
  count,
  viewAll,
  children,
}: {
  title: string;
  count?: number;
  viewAll?: Route;
  children: ReactNode;
}) {
  return (
    <div className="rounded-lg border border-grayA-4 bg-background">
      <div className="flex h-9 items-center justify-between gap-2 border-b border-grayA-4 px-3">
        <span className="flex min-w-0 items-center gap-2">
          <span className="truncate text-[13px] font-medium text-accent-12">{title}</span>
          {count !== undefined && <CountPill value={count} />}
        </span>
        {viewAll && (
          <Link
            href={viewAll}
            className="shrink-0 text-xs text-gray-9 transition-colors hover:text-accent-12"
          >
            View all
          </Link>
        )}
      </div>
      <div className="p-1">{children}</div>
    </div>
  );
}

function CardRow({ row, options }: { row: LaunchpadRow } & Pick<VariantProps, "options">) {
  return (
    <RowLink row={row} density={options.density}>
      <span className="min-w-0 flex-1 truncate text-accent-12">{row.name}</span>
      <RowSpark row={row} options={options} />
      <Value row={row} />
    </RowLink>
  );
}

/** Today's shape with the padding taken out: bordered cards, one row per resource. */
export function RailCards({ model, options }: VariantProps) {
  if (model.isLoading) {
    return <RowSkeleton density={options.density} />;
  }
  if (model.isEmpty || options.forceEmpty) {
    return (
      <Card title="Get started">
        <GetStarted density={options.density} />
      </Card>
    );
  }
  return (
    <>
      {model.keyspaces.length > 0 && (
        <Card title="Keyspaces" count={model.keyspaces.length} viewAll={model.keyspacesHref}>
          {model.keyspaces.slice(0, 5).map((row) => (
            <CardRow key={row.id} row={row} options={options} />
          ))}
        </Card>
      )}
      {model.ratelimits.length > 0 && (
        <Card title="Ratelimits" count={model.ratelimits.length} viewAll={model.ratelimitsHref}>
          {model.ratelimits.slice(0, 5).map((row) => (
            <CardRow key={row.id} row={row} options={options} />
          ))}
        </Card>
      )}
      {model.identityCount > 0 && (
        <Card title="Identities">
          <IdentitiesRow
            count={model.identityCount}
            href={model.identitiesHref}
            density={options.density}
          />
        </Card>
      )}
      <DeployNudge />
    </>
  );
}

/** Same cards, but each row keeps the meta line so nothing is lost to hover. */
export function RailCardsTwoLine({ model, options }: VariantProps) {
  if (model.isLoading) {
    return <RowSkeleton density="roomy" />;
  }
  if (model.isEmpty || options.forceEmpty) {
    return (
      <Card title="Get started">
        <GetStarted density={options.density} />
      </Card>
    );
  }
  const section = (title: string, rows: LaunchpadRow[], viewAll: Route) =>
    rows.length > 0 && (
      <Card title={title} count={rows.length} viewAll={viewAll}>
        {rows.slice(0, 4).map((row) => (
          <Link
            key={row.id}
            href={row.href}
            className={cn(
              "group flex items-center gap-2 rounded-md px-2 py-1.5 transition-colors hover:bg-grayA-2",
            )}
          >
            <span className="flex min-w-0 flex-1 flex-col">
              <span className="truncate text-[13px] text-accent-12">{row.name}</span>
              <RowMeta row={row} showProject={options.showProject} />
            </span>
            <RowSpark row={row} options={options} />
            <Value row={row} />
          </Link>
        ))}
      </Card>
    );
  return (
    <>
      {section("Keyspaces", model.keyspaces, model.keyspacesHref)}
      {section("Ratelimits", model.ratelimits, model.ratelimitsHref)}
      {model.identityCount > 0 && (
        <Card title="Identities">
          <IdentitiesRow
            count={model.identityCount}
            href={model.identitiesHref}
            density={options.density}
          />
        </Card>
      )}
    </>
  );
}

/** Three peer sections, no card chrome: a label, then rows on the page. */
export function RailPlain({ model, options }: VariantProps) {
  if (model.isLoading) {
    return <RowSkeleton density={options.density} />;
  }
  if (model.isEmpty || options.forceEmpty) {
    return (
      <section>
        <SectionLabel>Get started</SectionLabel>
        <GetStarted density={options.density} />
      </section>
    );
  }
  const section = (title: string, rows: LaunchpadRow[], viewAll: Route) =>
    rows.length > 0 && (
      <section>
        <SectionLabel action={{ label: "View all", href: viewAll }}>{title}</SectionLabel>
        {rows.slice(0, 5).map((row) => (
          <RowLink key={row.id} row={row} density={options.density}>
            <span className="min-w-0 flex-1 truncate text-accent-12">{row.name}</span>
            {options.showProject && row.projectName && (
              <span className="hidden shrink-0 truncate text-xs text-gray-9 @[240px]:inline">
                {row.projectName}
              </span>
            )}
            <RowSpark row={row} options={options} />
            <Value row={row} />
          </RowLink>
        ))}
      </section>
    );
  return (
    <>
      {section("Keyspaces", model.keyspaces, model.keyspacesHref)}
      {section("Ratelimits", model.ratelimits, model.ratelimitsHref)}
      {model.identityCount > 0 && (
        <section>
          <SectionLabel action={{ label: "View all", href: model.identitiesHref }}>
            Identities
          </SectionLabel>
          <div className={cn("flex items-center px-2 text-[13px]", ROW_HEIGHT[options.density])}>
            <span className="flex-1 text-gray-11">Total</span>
            <span className="tabular-nums text-accent-12">{fmt(model.identityCount)}</span>
          </div>
        </section>
      )}
      <DeployNudge compact />
    </>
  );
}
