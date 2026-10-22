"use client";

import { SourceIcon, appSource } from "@/components/deploy/app-source";
import { HighlightMatch } from "@/components/deploy/env-vars/highlight-match";
import { TYPE_MARKERS, TypeMarker } from "@/components/deploy/env-vars/type-marker";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { routes } from "@/lib/navigation/routes";
import { PaginationControls, ResourceListContent } from "@unkey/ui";
import type { Route } from "next";
import Link from "next/link";
import { useState } from "react";
import {
  PAGE_SIZE,
  type PageWindow,
  type VisibleAppGroup,
  type VisibleVariable,
  environmentProse,
  formatChangedAt,
  pageWindow,
} from "./model";

export function AppGroupCard({ group, search }: { group: VisibleAppGroup; search: string }) {
  const workspace = useWorkspaceNavigation();
  const [page, setPage] = useState(0);
  const { project, app, variables } = group;
  const pager = pageWindow(variables.length, page);
  const now = Date.now();
  const hrefFor = (key?: string) =>
    routes.projects.apps.envVars({
      workspaceSlug: workspace.slug,
      projectId: project.id,
      appId: app.id,
      search: key,
    });

  return (
    <ResourceListContent>
      <Link
        href={hrefFor()}
        className="flex h-11 items-center gap-2.5 border-b px-4 text-sm transition-colors hover:bg-grayA-2 focus-visible:bg-grayA-2 focus-visible:outline-hidden focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-grayA-7"
      >
        <SourceIcon source={appSource(app)} className="size-3.5 shrink-0 text-gray-12" />
        <span className="truncate font-medium text-gray-12">{app.name}</span>
        <span className="text-xs text-gray-9">{variables.length}</span>
      </Link>
      <ul aria-label={`Environment variables in ${app.name}`} className="py-1">
        {variables.slice(pager.start, pager.end).map((variable) => (
          <VariableRow
            key={variable.id}
            variable={variable}
            href={hrefFor(variable.key)}
            search={search}
            now={now}
          />
        ))}
      </ul>
      {variables.length > PAGE_SIZE && (
        <CardPagination
          label={`${app.name} pagination`}
          pager={pager}
          onPrevious={() => setPage(pager.current - 1)}
          onNext={() => setPage(pager.current + 1)}
        />
      )}
    </ResourceListContent>
  );
}

function CardPagination({
  label,
  pager,
  onPrevious,
  onNext,
}: {
  label: string;
  pager: PageWindow;
  onPrevious: () => void;
  onNext: () => void;
}) {
  return (
    <div className="flex h-10 items-center justify-between border-t pr-2 pl-4 text-xs text-grayA-9">
      <span className="tabular-nums">{pager.rangeLabel}</span>
      <PaginationControls
        label={label}
        hasPrevious={pager.hasPrevious}
        hasNext={pager.hasNext}
        onPrevious={onPrevious}
        onNext={onNext}
      />
    </div>
  );
}

function VariableRow({
  variable,
  href,
  search,
  now,
}: {
  variable: VisibleVariable;
  href: Route;
  search: string;
  now: number;
}) {
  return (
    <li>
      <Link
        href={href}
        className="grid h-8 grid-cols-[minmax(0,1.5fr)_minmax(0,1.5fr)_minmax(0,1fr)_64px] items-center gap-4 px-4 text-xs transition-colors hover:bg-grayA-2 focus-visible:bg-grayA-2 focus-visible:outline-hidden focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-grayA-7"
      >
        <span className="truncate font-mono text-gray-12">
          <HighlightMatch text={variable.key} query={search} />
        </span>
        <span className="truncate text-gray-11">{environmentProse(variable.environments)}</span>
        <span className="flex min-w-0 items-center gap-1.5 text-gray-11">
          <span aria-hidden="true" className="contents">
            <TypeMarker type={variable.type} className="size-5 shrink-0" />
          </span>
          <span className="truncate">{TYPE_MARKERS[variable.type].name}</span>
        </span>
        <span className="text-right text-gray-11 tabular-nums">
          {formatChangedAt(variable.changedAt, now)}
        </span>
      </Link>
    </li>
  );
}
