"use client";

import { ListPager } from "@/components/list-pager";
import { LoadError } from "@/components/load-error";
import { useListView } from "@/hooks/use-list-view";
import type { SearchedListView } from "@/lib/loadable/list";
import { IconMagnifierOutline18 } from "@unkey/icons";
import { match } from "@unkey/match";
import {
  EmptyState,
  EmptyStateDescription,
  EmptyStateHeader,
  EmptyStateIcon,
  EmptyStateTitle,
  ResourceList,
  ResourceListBody,
  type ResourceListColumn,
  ResourceListContent,
  ResourceListGrid,
  ResourceListHeader,
  ResourceListTable,
  ResourceListTableRow,
  type ResourceListView,
  ResourceListViewToggle,
  Skeleton,
} from "@unkey/ui";
import type { Route } from "next";
import Link from "next/link";
import { type ComponentProps, Fragment, type ReactNode } from "react";

type ListColumnSkeleton = "title" | "text" | "none";

export type ListColumn<Id extends string> = ResourceListColumn & {
  id: Id;
  skeleton: ListColumnSkeleton;
};

export type ListTable<Id extends string> = {
  columns: readonly ListColumn<Id>[];
  minWidth: number;
};

export function listTable<const Id extends string>(table: ListTable<Id>): ListTable<Id> {
  return table;
}

type ShownList<Row> = Extract<SearchedListView<Row>, { status: "loading" | "ready" }>;

type ListLayout<Row, Id extends string> = {
  noun: string;
  table: ListTable<Id>;
  rowKey: (row: Row) => string;
  renderCard: (row: Row) => ReactNode;
  renderRow: (row: Row) => ReactNode;
  cardSkeleton: ReactNode;
};

type ListViewsProps<Row, Id extends string> = ListLayout<Row, Id> & {
  list: SearchedListView<Row>;
  viewStorageKey: string;
  controls: ReactNode;
  banner?: ReactNode;
  empty: ReactNode;
  failedTitle: string;
};

const SKELETON_KEYS = ["skeleton-1", "skeleton-2", "skeleton-3"];

const SKELETON_CELLS: Record<ListColumnSkeleton, ReactNode> = {
  title: <Skeleton className="h-3 w-28" />,
  text: <Skeleton className="h-3 w-16" />,
  none: <span />,
};

export function ListViews<Row, Id extends string>({
  list,
  viewStorageKey,
  controls,
  banner,
  empty,
  failedTitle,
  ...layout
}: ListViewsProps<Row, Id>) {
  const [view, setView] = useListView(viewStorageKey);
  const shell = (body: ReactNode) => (
    <ResourceList>
      {banner}
      <ResourceListHeader>
        {controls}
        <ResourceListViewToggle className="md:ml-auto" value={view} onValueChange={setView} />
      </ResourceListHeader>
      {body}
    </ResourceList>
  );

  return match(list)
    .with({ status: "empty" }, () => empty)
    .with({ status: "failed" }, ({ error, retry }) =>
      shell(<LoadError title={failedTitle} error={error} onRetry={retry} />),
    )
    .with({ status: "noMatch" }, ({ search }) =>
      shell(<ListNoMatch noun={layout.noun} search={search} />),
    )
    .with({ status: "loading" }, { status: "ready" }, (shown) =>
      shell(<ListLayoutView view={view} list={shown} {...layout} />),
    )
    .exhaustive();
}

function ListLayoutView<Row, Id extends string>({
  view,
  ...props
}: ListLayout<Row, Id> & { view: ResourceListView; list: ShownList<Row> }) {
  return match(view)
    .with("grid", () => <ListGrid {...props} />)
    .with("list", () => <ListTableView {...props} />)
    .exhaustive();
}

function ListNoMatch({ noun, search }: { noun: string; search: string }) {
  return (
    <ResourceListContent>
      <EmptyState frame="none">
        <EmptyStateIcon>
          <IconMagnifierOutline18 />
        </EmptyStateIcon>
        <EmptyStateHeader>
          <EmptyStateTitle>No {noun} found</EmptyStateTitle>
          <EmptyStateDescription>
            No {noun} match "{search}". Try a different search term.
          </EmptyStateDescription>
        </EmptyStateHeader>
      </EmptyState>
    </ResourceListContent>
  );
}

function ListGrid<Row>({
  list,
  noun,
  rowKey,
  renderCard,
  cardSkeleton,
}: Pick<ListLayout<Row, string>, "noun" | "rowKey" | "renderCard" | "cardSkeleton"> & {
  list: ShownList<Row>;
}) {
  return match(list)
    .with({ status: "loading" }, () => (
      <div aria-busy="true">
        <output className="sr-only">Loading {noun}...</output>
        <ResourceListGrid aria-hidden="true">
          {SKELETON_KEYS.map((key) => (
            <Fragment key={key}>{cardSkeleton}</Fragment>
          ))}
        </ResourceListGrid>
      </div>
    ))
    .with({ status: "ready" }, ({ rows, footer }) => (
      <>
        <ResourceListGrid>
          {rows.map((row) => (
            <Fragment key={rowKey(row)}>{renderCard(row)}</Fragment>
          ))}
        </ResourceListGrid>
        <ListPager state={footer} noun={noun} variant="bare" />
      </>
    ))
    .exhaustive();
}

function ListTableView<Row, Id extends string>({
  list,
  noun,
  table,
  rowKey,
  renderRow,
}: Pick<ListLayout<Row, Id>, "noun" | "table" | "rowKey" | "renderRow"> & {
  list: ShownList<Row>;
}) {
  return match(list)
    .with({ status: "loading" }, () => (
      <ResourceListTable columns={table.columns} minWidth={table.minWidth} aria-busy="true">
        <output className="sr-only">Loading {noun}...</output>
        <ResourceListBody aria-hidden="true">
          {SKELETON_KEYS.map((key) => (
            <ResourceListTableRow key={key}>
              {table.columns.map((column) => (
                <Fragment key={column.id}>{SKELETON_CELLS[column.skeleton]}</Fragment>
              ))}
            </ResourceListTableRow>
          ))}
        </ResourceListBody>
      </ResourceListTable>
    ))
    .with({ status: "ready" }, ({ rows, footer }) => (
      <ResourceListTable
        columns={table.columns}
        minWidth={table.minWidth}
        footer={<ListPager state={footer} noun={noun} />}
      >
        <ResourceListBody>
          {rows.map((row) => (
            <Fragment key={rowKey(row)}>{renderRow(row)}</Fragment>
          ))}
        </ResourceListBody>
      </ResourceListTable>
    ))
    .exhaustive();
}

export function ListTableRow<Id extends string>({
  table,
  cells,
  href,
  label,
  ...props
}: Omit<ComponentProps<"li">, "children"> & {
  table: ListTable<Id>;
  cells: Record<Id, ReactNode>;
  href: Route;
  label: string;
}) {
  return (
    <ResourceListTableRow link={<Link href={href} aria-label={label} />} {...props}>
      {table.columns.map((column) => (
        <Fragment key={column.id}>{cells[column.id]}</Fragment>
      ))}
    </ResourceListTableRow>
  );
}
