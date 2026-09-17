"use client";

import { cn } from "@/lib/utils";
import { IconFingerprintOutline18 } from "@unkey/icons";
import Link from "next/link";
import { KindGlyph, ROW_HEIGHT, RowLink, RowSkeleton, SectionLabel, fmt } from "../parts";
import type { VariantProps } from "../types";
import { DeployNudge, GetStarted, RowSpark, Value } from "./shared";

/** Columns line up, so scanning down the numbers is the fast path. */
export function RailTable({ model, options }: VariantProps) {
  if (model.isLoading) {
    return <RowSkeleton density={options.density} count={7} />;
  }
  if (model.isEmpty || options.forceEmpty) {
    return (
      <section>
        <SectionLabel>Get started</SectionLabel>
        <GetStarted density={options.density} />
      </section>
    );
  }
  const showProject = options.showProject && model.projectCount > 1;
  return (
    <>
      <div className="overflow-hidden rounded-lg border border-grayA-4">
        <div className="flex h-7 items-center gap-2 border-b border-grayA-4 bg-grayA-2 px-2 text-[11px] uppercase tracking-wide text-gray-9">
          <span className="w-3" />
          <span className="min-w-0 flex-1">Resource</span>
          {showProject && <span className="w-20 shrink-0">Project</span>}
          <span className="w-12 shrink-0 text-right">{`${model.windowHours}h`}</span>
        </div>
        {model.rows.slice(0, 9).map((row) => (
          <RowLink key={row.id} row={row} density={options.density} className="rounded-none">
            <KindGlyph kind={row.kind} />
            <span className="min-w-0 flex-1 truncate text-accent-12">{row.name}</span>
            {showProject && (
              <span className="w-20 shrink-0 truncate text-xs text-gray-9">
                {row.projectName ?? "—"}
              </span>
            )}
            <RowSpark row={row} options={options} />
            <Value row={row} className="w-12 text-right" />
          </RowLink>
        ))}
        <Link
          href={model.identitiesHref}
          className={cn(
            "flex items-center gap-2 border-t border-grayA-4 px-2 text-[13px] transition-colors hover:bg-grayA-2",
            ROW_HEIGHT[options.density],
          )}
        >
          <IconFingerprintOutline18 className="size-3 shrink-0 text-gray-9" />
          <span className="flex-1 text-gray-11">Identities</span>
          <span className="tabular-nums text-accent-12">{fmt(model.identityCount)}</span>
        </Link>
      </div>
      <DeployNudge compact />
    </>
  );
}

/** Narrow column: the name and the number, nothing competing for the width. */
export function RailNarrow({ model, options }: VariantProps) {
  if (model.isLoading) {
    return <RowSkeleton density={options.density} count={7} />;
  }
  if (model.isEmpty || options.forceEmpty) {
    return <GetStarted density={options.density} />;
  }
  return (
    <>
      <section>
        <SectionLabel>{`Most used · ${model.windowHours}h`}</SectionLabel>
        {model.rows.slice(0, 9).map((row) => (
          <RowLink key={row.id} row={row} density={options.density}>
            <KindGlyph kind={row.kind} />
            <span className="min-w-0 flex-1 truncate text-accent-12">{row.name}</span>
            <Value row={row} />
          </RowLink>
        ))}
      </section>
      {model.identityCount > 0 && (
        <div
          className={cn("flex items-center px-2 text-[13px]", ROW_HEIGHT[options.density])}
        >
          <span className="flex-1 text-gray-9">Identities</span>
          <span className="tabular-nums text-gray-11">{fmt(model.identityCount)}</span>
        </div>
      )}
    </>
  );
}

/** Not a rail at all: a band under the grid so projects own the full width. */
export function BandColumns({ model, options }: VariantProps) {
  if (model.isLoading) {
    return <RowSkeleton density={options.density} count={4} />;
  }
  if (model.isEmpty || options.forceEmpty) {
    return (
      <section>
        <SectionLabel>Get started</SectionLabel>
        <GetStarted density={options.density} />
      </section>
    );
  }
  const column = (title: string, rows: typeof model.rows, viewAll: VariantProps["model"]["identitiesHref"]) => (
    <section className="min-w-0">
      <SectionLabel action={{ label: "View all", href: viewAll }}>{title}</SectionLabel>
      {rows.slice(0, 4).map((row) => (
        <RowLink key={row.id} row={row} density={options.density}>
          <span className="min-w-0 flex-1 truncate text-accent-12">{row.name}</span>
          <RowSpark row={row} options={options} />
          <Value row={row} />
        </RowLink>
      ))}
      {rows.length === 0 && (
        <div className="px-2 py-2 text-xs text-gray-9">Nothing here yet</div>
      )}
    </section>
  );
  return (
    <div className="grid grid-cols-1 gap-x-8 gap-y-4 md:grid-cols-3">
      {column("Keyspaces", model.keyspaces, model.keyspacesHref)}
      {column("Ratelimits", model.ratelimits, model.ratelimitsHref)}
      <section className="min-w-0">
        <SectionLabel action={{ label: "View all", href: model.identitiesHref }}>
          Identities
        </SectionLabel>
        <div className={cn("flex items-center px-2 text-[13px]", ROW_HEIGHT[options.density])}>
          <span className="flex-1 text-gray-11">Total</span>
          <span className="tabular-nums text-accent-12">{fmt(model.identityCount)}</span>
        </div>
        <DeployNudge compact />
      </section>
    </div>
  );
}
