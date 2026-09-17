"use client";

import { cn } from "@/lib/utils";
import { IconFingerprintOutline18, IconMagnifierOutline12 } from "@unkey/icons";
import Link from "next/link";
import { useMemo, useState } from "react";
import { IdentitiesRow, KindGlyph, ROW_HEIGHT, RowLink, RowSkeleton, SectionLabel, fmt } from "../parts";
import type { VariantProps } from "../types";
import { DeployNudge, GetStarted, RowMeta, RowSpark, Value } from "./shared";

/** One ranked list across both products. The glyph carries the type, not colour. */
export function RailFlat({ model, options }: VariantProps) {
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
  return (
    <>
      <section>
        <SectionLabel>{`Most used · ${model.windowHours}h`}</SectionLabel>
        {model.rows.slice(0, 8).map((row) => (
          <RowLink key={row.id} row={row} density={options.density}>
            <KindGlyph kind={row.kind} />
            <span className="min-w-0 flex-1 truncate text-accent-12">{row.name}</span>
            {options.showProject && row.projectName && model.projectCount > 1 && (
              <span className="shrink-0 truncate text-xs text-gray-9">{row.projectName}</span>
            )}
            <RowSpark row={row} options={options} />
            <Value row={row} />
          </RowLink>
        ))}
      </section>
      {model.identityCount > 0 && (
        <IdentitiesRow
          count={model.identityCount}
          href={model.identitiesHref}
          density={options.density}
        />
      )}
      <DeployNudge compact />
    </>
  );
}

/** Ranked list behind a filter, for a default project holding thousands of keyspaces. */
export function RailSearch({ model, options }: VariantProps) {
  const [term, setTerm] = useState("");
  const filtered = useMemo(() => {
    const needle = term.trim().toLowerCase();
    if (!needle) {
      return model.rows;
    }
    return model.rows.filter((row) => row.name.toLowerCase().includes(needle));
  }, [model.rows, term]);

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
  return (
    <div className="rounded-lg border border-grayA-4 bg-background">
      <label className="flex h-9 items-center gap-2 border-b border-grayA-4 px-3">
        <IconMagnifierOutline12 className="size-3 shrink-0 text-gray-9" />
        <input
          value={term}
          onChange={(event) => setTerm(event.target.value)}
          placeholder="Jump to a keyspace or ratelimit"
          className="min-w-0 flex-1 bg-transparent text-[13px] text-accent-12 outline-none placeholder:text-gray-9"
        />
        <span className="shrink-0 tabular-nums text-[11px] text-gray-9">{filtered.length}</span>
      </label>
      <div className="max-h-[360px] overflow-y-auto p-1">
        {filtered.slice(0, 40).map((row) => (
          <RowLink key={row.id} row={row} density={options.density}>
            <KindGlyph kind={row.kind} />
            <span className="min-w-0 flex-1 truncate text-accent-12">{row.name}</span>
            <RowSpark row={row} options={options} />
            <Value row={row} />
          </RowLink>
        ))}
        {filtered.length === 0 && (
          <div className="px-2 py-6 text-center text-xs text-gray-9">Nothing matches “{term}”</div>
        )}
      </div>
      <Link
        href={model.identitiesHref}
        className="flex h-9 items-center gap-2 border-t border-grayA-4 px-3 text-[13px] text-gray-11 transition-colors hover:bg-grayA-2"
      >
        <IconFingerprintOutline18 className="size-3.5 shrink-0 text-gray-9" />
        <span className="flex-1">Identities</span>
        <span className="tabular-nums text-accent-12">{fmt(model.identityCount)}</span>
      </Link>
    </div>
  );
}

/** Totals first; a section opens only when you want the names behind it. */
export function RailSummary({ model, options }: VariantProps) {
  const [open, setOpen] = useState<"keyspace" | "ratelimit" | null>("keyspace");

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

  const group = (kind: "keyspace" | "ratelimit", label: string) => {
    const rows = kind === "keyspace" ? model.keyspaces : model.ratelimits;
    if (rows.length === 0) {
      return null;
    }
    const total = rows.reduce((sum, row) => sum + row.total, 0);
    const isOpen = open === kind;
    return (
      <div>
        <button
          type="button"
          onClick={() => setOpen(isOpen ? null : kind)}
          className={cn(
            "flex w-full items-center gap-2 rounded-md px-2 text-[13px] transition-colors hover:bg-grayA-2",
            ROW_HEIGHT[options.density],
          )}
        >
          <KindGlyph kind={kind} />
          <span className="flex-1 text-left text-accent-12">{label}</span>
          <span className="text-xs tabular-nums text-gray-9">{rows.length}</span>
          <span className="tabular-nums text-accent-12">{fmt(total)}</span>
        </button>
        {isOpen && (
          <div className="ml-4 border-l border-grayA-4 pl-1">
            {rows.slice(0, 5).map((row) => (
              <RowLink key={row.id} row={row} density="compact">
                <span className="min-w-0 flex-1 truncate text-gray-11">{row.name}</span>
                <RowSpark row={row} options={options} />
                <Value row={row} className="text-gray-11" />
              </RowLink>
            ))}
          </div>
        )}
      </div>
    );
  };

  return (
    <>
      <section>
        <SectionLabel>{`Activity · ${model.windowHours}h`}</SectionLabel>
        {group("keyspace", "Keyspaces")}
        {group("ratelimit", "Ratelimits")}
        <IdentitiesRow
          count={model.identityCount}
          href={model.identitiesHref}
          density={options.density}
        />
      </section>
      <DeployNudge compact />
    </>
  );
}

/** Quietest reading: hairlines, dimmed names, numbers doing the talking. */
export function RailQuiet({ model, options }: VariantProps) {
  if (model.isLoading) {
    return <RowSkeleton density={options.density} count={7} />;
  }
  if (model.isEmpty || options.forceEmpty) {
    return <GetStarted density={options.density} />;
  }
  return (
    <div className="flex flex-col">
      <span className="px-2 pb-1 text-[11px] font-medium uppercase tracking-wide text-gray-8">
        {`Most used · ${model.windowHours}h`}
      </span>
      <div className="divide-y divide-grayA-3">
        {model.rows.slice(0, 8).map((row) => (
          <RowLink key={row.id} row={row} density={options.density} className="rounded-none">
            <span className="min-w-0 flex-1 truncate text-gray-11">{row.name}</span>
            <RowMeta row={row} showProject={options.showProject} className="shrink-0" />
            <RowSpark row={row} options={options} />
            <span className="shrink-0 tabular-nums text-gray-11">{fmt(row.total)}</span>
          </RowLink>
        ))}
      </div>
      {model.identityCount > 0 && (
        <div
          className={cn(
            "flex items-center border-t border-grayA-3 px-2 text-[13px]",
            ROW_HEIGHT[options.density],
          )}
        >
          <span className="flex-1 text-gray-9">Identities</span>
          <span className="tabular-nums text-gray-11">{fmt(model.identityCount)}</span>
        </div>
      )}
    </div>
  );
}
