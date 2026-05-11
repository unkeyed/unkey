"use client";

import { BrandLogo } from "@/lib/extensions/brand-logos";
import { useInstallations } from "@/lib/extensions/installations";
import {
  ALL_CATEGORIES,
  CATEGORY_LABELS,
  EXTENSIONS,
  EXTENSION_TYPE_LABELS,
  type Extension,
  type ExtensionCategory,
  type ExtensionType,
} from "@/lib/extensions/registry";
import { ArrowRight, CircleCheck, Magnifier } from "@unkey/icons";
import { Button, Input } from "@unkey/ui";
import { cn } from "@unkey/ui/src/lib/utils";
import Link from "next/link";
import { useMemo, useState } from "react";
import { ExtensionCard } from "./components/extension-card";

type SortKey = "popular" | "recent" | "alphabetical";
type TypeFilter = "all" | ExtensionType;

type ExtensionsClientProps = {
  basePath: string;
  projectId: string;
};

export function ExtensionsClient({ basePath, projectId }: ExtensionsClientProps) {
  const [search, setSearch] = useState("");
  const [type, setType] = useState<TypeFilter>("all");
  const [sort, setSort] = useState<SortKey>("popular");
  const [selectedCategories, setSelectedCategories] = useState<ExtensionCategory[]>([]);

  const { installations } = useInstallations(projectId);
  const installedSlugs = useMemo(
    () => new Set(installations.map((i) => i.extensionSlug)),
    [installations],
  );

  const counts = useMemo(() => buildCategoryCounts(EXTENSIONS), []);

  const filtered = useMemo(() => {
    return EXTENSIONS.filter((extension) => {
      if (type !== "all" && extension.type !== type) {
        return false;
      }
      if (
        selectedCategories.length > 0 &&
        !extension.categories.some((c) => selectedCategories.includes(c))
      ) {
        return false;
      }
      if (search.trim().length > 0) {
        const needle = search.trim().toLowerCase();
        const haystack =
          `${extension.name} ${extension.tagline} ${extension.categories.join(" ")}`.toLowerCase();
        if (!haystack.includes(needle)) {
          return false;
        }
      }
      return true;
    }).sort(sortComparator(sort));
  }, [search, type, selectedCategories, sort]);

  const isFilterActive =
    search.trim().length > 0 || type !== "all" || selectedCategories.length > 0;

  const featured = useMemo(() => EXTENSIONS.find((e) => e.featured), []);

  return (
    <div className="flex flex-col gap-8">
      {!isFilterActive && featured ? (
        <FeaturedHero
          extension={featured}
          href={`${basePath}/${featured.slug}`}
          installed={installedSlugs.has(featured.slug)}
        />
      ) : null}

      <div className="flex flex-col gap-5">
        <Toolbar
          search={search}
          onSearchChange={setSearch}
          type={type}
          onTypeChange={setType}
          sort={sort}
          onSortChange={setSort}
        />

        <CategoryChips
          counts={counts}
          totalCount={EXTENSIONS.length}
          selected={selectedCategories}
          onToggle={(c) =>
            setSelectedCategories((prev) =>
              prev.includes(c) ? prev.filter((x) => x !== c) : [...prev, c],
            )
          }
          onClear={() => setSelectedCategories([])}
        />
      </div>

      {filtered.length === 0 ? (
        <EmptyState />
      ) : (
        <div className="flex flex-col gap-3">
          <div className="flex items-baseline justify-between">
            <h2 className="font-semibold text-gray-12 text-sm">
              {isFilterActive
                ? `${filtered.length} match${filtered.length === 1 ? "" : "es"}`
                : "All extensions"}
            </h2>
            <span className="text-[11px] text-gray-10 font-mono uppercase tracking-wide">
              {filtered.length} of {EXTENSIONS.length}
            </span>
          </div>
          <ExtensionGrid
            extensions={filtered}
            basePath={basePath}
            installedSlugs={installedSlugs}
          />
        </div>
      )}
    </div>
  );
}

function FeaturedHero({
  extension,
  href,
  installed,
}: {
  extension: Extension;
  href: string;
  installed: boolean;
}) {
  return (
    <Link
      href={href}
      className="group relative overflow-hidden rounded-2xl border border-grayA-4 bg-gradient-to-br from-grayA-2 via-background to-background p-6 transition-all duration-300 hover:border-grayA-7"
    >
      {/* subtle accent in the corner */}
      <div
        aria-hidden
        className="pointer-events-none absolute -right-24 -top-24 size-64 rounded-full bg-accent-3 blur-3xl opacity-50"
      />

      <div className="relative flex flex-col gap-5 md:flex-row md:items-center md:justify-between">
        <div className="flex items-start gap-4 min-w-0">
          <div className="size-14 bg-white rounded-2xl flex items-center justify-center shrink-0 shadow-sm shadow-grayA-8/30 ring-1 ring-grayA-3 overflow-hidden">
            <BrandLogo
              slug={extension.slug}
              iconUrl={extension.iconUrl}
              name={extension.name}
              className="size-9"
            />
          </div>
          <div className="flex flex-col gap-1.5 min-w-0">
            <div className="flex items-center gap-2">
              <span className="text-[11px] font-mono uppercase tracking-wide text-accent-11">
                Featured
              </span>
              <span className="text-[11px] text-gray-10">·</span>
              <span className="text-[11px] font-mono uppercase tracking-wide text-gray-10">
                {EXTENSION_TYPE_LABELS[extension.type]}
              </span>
              {installed ? (
                <span className="rounded-md bg-successA-3 px-1.5 py-0.5 text-[10px] font-medium uppercase tracking-wide text-successA-11">
                  Installed
                </span>
              ) : null}
            </div>
            <h2 className="text-xl font-semibold text-accent-12 leading-7">{extension.name}</h2>
            <p className="text-[13px] leading-5 text-gray-11 max-w-xl">{extension.tagline}</p>
          </div>
        </div>

        <div className="flex items-center gap-2 shrink-0">
          <Button variant="primary">
            View extension
            <ArrowRight className="size-3.5" />
          </Button>
        </div>
      </div>
    </Link>
  );
}

function Toolbar({
  search,
  onSearchChange,
  type,
  onTypeChange,
  sort,
  onSortChange,
}: {
  search: string;
  onSearchChange: (v: string) => void;
  type: TypeFilter;
  onTypeChange: (v: TypeFilter) => void;
  sort: SortKey;
  onSortChange: (v: SortKey) => void;
}) {
  return (
    <div className="flex flex-col gap-3 md:flex-row md:items-center">
      <div className="flex-1 max-w-md">
        <Input
          value={search}
          onChange={(e) => onSearchChange(e.target.value)}
          placeholder="Search extensions"
          leftIcon={<Magnifier />}
        />
      </div>

      <div className="flex items-center gap-2 ml-auto">
        <Pill active={type === "all"} onClick={() => onTypeChange("all")}>
          All
        </Pill>
        <Pill active={type === "native"} onClick={() => onTypeChange("native")}>
          Native
        </Pill>
        <Pill active={type === "partner"} onClick={() => onTypeChange("partner")}>
          Partner
        </Pill>
        <Pill active={type === "community"} onClick={() => onTypeChange("community")}>
          Community
        </Pill>

        <div className="ml-2 flex items-center">
          <select
            value={sort}
            onChange={(e) => onSortChange(e.target.value as SortKey)}
            className="h-8 rounded-md border border-grayA-4 bg-background px-2 text-[12px] text-gray-12 hover:border-grayA-6 focus:outline-none focus:ring-2 focus:ring-grayA-5"
            aria-label="Sort extensions"
          >
            <option value="popular">Popular</option>
            <option value="recent">Recent</option>
            <option value="alphabetical">A–Z</option>
          </select>
        </div>
      </div>
    </div>
  );
}

function Pill({
  active,
  onClick,
  children,
}: {
  active: boolean;
  onClick: () => void;
  children: React.ReactNode;
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      className={cn(
        "h-8 rounded-md border px-2.5 text-[12px] font-medium transition-colors",
        active
          ? "border-grayA-7 bg-grayA-3 text-accent-12"
          : "border-grayA-4 bg-background text-gray-11 hover:border-grayA-6 hover:text-gray-12",
      )}
    >
      {children}
    </button>
  );
}

function CategoryChips({
  counts,
  totalCount,
  selected,
  onToggle,
  onClear,
}: {
  counts: Record<ExtensionCategory, number>;
  totalCount: number;
  selected: ExtensionCategory[];
  onToggle: (c: ExtensionCategory) => void;
  onClear: () => void;
}) {
  const visible = ALL_CATEGORIES.filter((c) => (counts[c] ?? 0) > 0);

  return (
    <div className="flex flex-wrap items-center gap-2">
      <Chip active={selected.length === 0} onClick={onClear} count={totalCount}>
        All
      </Chip>
      {visible.map((category) => (
        <Chip
          key={category}
          active={selected.includes(category)}
          onClick={() => onToggle(category)}
          count={counts[category]}
        >
          {CATEGORY_LABELS[category]}
        </Chip>
      ))}
    </div>
  );
}

function Chip({
  active,
  onClick,
  count,
  children,
}: {
  active: boolean;
  onClick: () => void;
  count: number;
  children: React.ReactNode;
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      className={cn(
        "inline-flex h-7 items-center gap-1.5 rounded-full border px-2.5 text-[12px] transition-colors",
        active
          ? "border-grayA-7 bg-grayA-3 text-accent-12 font-medium"
          : "border-grayA-4 bg-background text-gray-11 hover:border-grayA-6 hover:text-gray-12",
      )}
    >
      {children}
      <span className={cn("tabular-nums text-[10px]", active ? "text-gray-11" : "text-gray-10")}>
        {count}
      </span>
      {active ? <CircleCheck className="size-3 text-accent-11" /> : null}
    </button>
  );
}

function ExtensionGrid({
  extensions,
  basePath,
  installedSlugs,
}: {
  extensions: Extension[];
  basePath: string;
  installedSlugs: Set<string>;
}) {
  return (
    <div className="grid grid-cols-1 gap-3 sm:grid-cols-2 xl:grid-cols-3">
      {extensions.map((extension) => (
        <ExtensionCard
          key={extension.slug}
          extension={extension}
          href={`${basePath}/${extension.slug}`}
          installed={installedSlugs.has(extension.slug)}
        />
      ))}
    </div>
  );
}

function EmptyState() {
  return (
    <div className="flex h-40 items-center justify-center rounded-2xl border border-dashed border-grayA-4 text-[13px] text-gray-10">
      No extensions match your filters.
    </div>
  );
}

function buildCategoryCounts(extensions: Extension[]): Record<ExtensionCategory, number> {
  const empty = Object.fromEntries(ALL_CATEGORIES.map((c) => [c, 0])) as Record<
    ExtensionCategory,
    number
  >;

  return extensions.reduce((acc, extension) => {
    for (const category of extension.categories) {
      acc[category] = (acc[category] ?? 0) + 1;
    }
    return acc;
  }, empty);
}

function sortComparator(sort: SortKey) {
  return (a: Extension, b: Extension): number => {
    switch (sort) {
      case "popular":
        return b.installs - a.installs;
      case "alphabetical":
        return a.name.localeCompare(b.name);
      case "recent": {
        const aDate = a.changelog.at(0)?.date ?? "";
        const bDate = b.changelog.at(0)?.date ?? "";
        return bDate.localeCompare(aDate);
      }
    }
  };
}
