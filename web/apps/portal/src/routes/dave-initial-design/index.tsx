import { createFileRoute, useNavigate } from "@tanstack/react-router";
import type { SortingState } from "@tanstack/react-table";
import { useMemo, useState } from "react";
import { z } from "zod";
import { KeysTable } from "~/components/keys-table/keys-table";
import type { StatusFilter } from "~/components/keys-table/keys-toolbar";
import { PortalFooter } from "~/components/portal-footer";
import { PortalLogoHeader } from "~/components/portal-logo-header";
import { type DemoState, PreviewSwitcher } from "./-preview-switcher";
import { type Key, seedBranding, seedKeys, synthesizeKeys } from "./-seed";

type CSSVarStyle = React.CSSProperties & { [K in `--${string}`]?: string };

const SORT_VALUES = [
  "none",
  "createdAt.desc",
  "createdAt.asc",
  "name.asc",
  "name.desc",
  "status.asc",
  "status.desc",
  "expires.asc",
  "expires.desc",
] as const;
type SortValue = (typeof SORT_VALUES)[number];

// Schema fields are optional with undefined fallback. Missing URL params stay
// undefined (not backfilled with defaults), so the router doesn't need to
// normalize `/dave-initial-design` → `/dave-initial-design?demo=few&q=&...`.
const searchSchema = z.object({
  demo: z.enum(["empty", "p50", "p99", "max"]).optional().catch(undefined),
  q: z.string().optional().catch(undefined),
  status: z.enum(["all", "enabled", "disabled", "expired"]).optional().catch(undefined),
  page: z.coerce.number().int().nonnegative().optional().catch(undefined),
  sort: z.enum(SORT_VALUES).optional().catch(undefined),
});

type SearchInput = z.infer<typeof searchSchema>;

type SearchResolved = {
  demo: DemoState;
  q: string;
  status: StatusFilter;
  page: number;
  sort: SortValue;
};

const DEFAULTS: SearchResolved = {
  demo: "p50",
  q: "",
  status: "all",
  page: 0,
  sort: "none",
};

function resolve(input: SearchInput): SearchResolved {
  return {
    demo: input.demo ?? DEFAULTS.demo,
    q: input.q ?? DEFAULTS.q,
    status: input.status ?? DEFAULTS.status,
    page: input.page ?? DEFAULTS.page,
    sort: input.sort ?? DEFAULTS.sort,
  };
}

function toUrl(resolved: SearchResolved): SearchInput {
  return {
    demo: resolved.demo === DEFAULTS.demo ? undefined : resolved.demo,
    q: resolved.q === DEFAULTS.q ? undefined : resolved.q,
    status: resolved.status === DEFAULTS.status ? undefined : resolved.status,
    page: resolved.page === DEFAULTS.page ? undefined : resolved.page,
    sort: resolved.sort === DEFAULTS.sort ? undefined : resolved.sort,
  };
}

export const Route = createFileRoute("/dave-initial-design/")({
  validateSearch: searchSchema,
  component: Preview,
});

function parseSort(s: SortValue): SortingState {
  if (s === "none") return [];
  const [id, dir] = s.split(".");
  return [{ id: id as string, desc: dir === "desc" }];
}

function stringifySort(state: SortingState, fallback: SortValue): SortValue {
  const head = state[0];
  if (!head) return "none";
  const v = `${head.id}.${head.desc ? "desc" : "asc"}` as SortValue;
  return (SORT_VALUES as readonly string[]).includes(v) ? v : fallback;
}

function Preview() {
  const search = Route.useSearch();
  const navigate = useNavigate({ from: "/dave-initial-design/" });
  const [deletedIds, setDeletedIds] = useState<Set<string>>(() => new Set());

  const resolved = useMemo(() => resolve(search), [search]);
  const sorting = useMemo(() => parseSort(resolved.sort), [resolved.sort]);
  const source = useMemo(
    () => resolveKeys(resolved.demo, resolved.page),
    [resolved.demo, resolved.page],
  );
  const keys = useMemo(
    () => source.keys.filter((k) => !deletedIds.has(k.id)),
    [source.keys, deletedIds],
  );

  const update = (next: Partial<SearchResolved>) => {
    navigate({
      search: (prev) => toUrl({ ...resolve(prev), ...next }),
      replace: true,
    });
  };

  const rootStyle: CSSVarStyle = {
    "--portal-bg": seedBranding.backgroundColor,
    "--portal-btn": seedBranding.buttonColor,
  };

  return (
    <div style={rootStyle} className="flex min-h-screen flex-col bg-background">
      <PortalLogoHeader branding={seedBranding} />
      <main className="flex-1">
        <div className="mx-auto max-w-5xl px-8 pt-8 pb-12">
          <KeysTable
            appName={seedBranding.appName}
            keys={keys}
            totalCount={source.totalCount}
            manualPagination={source.manualPagination}
            searchValue={resolved.q}
            onSearchChange={(q) => update({ q })}
            statusValue={resolved.status}
            onStatusChange={(status) => update({ status })}
            sorting={sorting}
            onSortingChange={(updater) => {
              const next = typeof updater === "function" ? updater(sorting) : updater;
              update({ sort: stringifySort(next, resolved.sort) });
            }}
            pageIndex={resolved.page}
            onPageChange={(page) => update({ page })}
            onDelete={(id) => setDeletedIds((prev) => new Set(prev).add(id))}
            onEditExpiration={(id) => console.log("edit", id)}
            onRotate={(id) => console.log("rotate", id)}
          />
        </div>
      </main>
      <PortalFooter />
      <PreviewSwitcher
        value={resolved.demo}
        onSelect={(demo) => update({ demo, page: 0, q: "", status: "all" })}
      />
    </div>
  );
}

function resolveKeys(
  demo: DemoState,
  page: number,
): { keys: Key[]; totalCount?: number; manualPagination?: boolean } {
  switch (demo) {
    case "empty":
      return { keys: [] };
    case "p50":
      return { keys: seedKeys.slice(0, 3) };
    case "p99":
      return { keys: seedKeys.slice(0, 7) };
    case "max":
      return {
        keys: synthesizeKeys({ page }),
        totalCount: 37_566,
        manualPagination: true,
      };
  }
}
