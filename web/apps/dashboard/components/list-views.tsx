"use client";

import { ListPager } from "@/components/list-pager";
import type { ListContent } from "@/lib/loadable/list";
import { IconMagnifierOutline18 } from "@unkey/icons";
import { match } from "@unkey/match";
import {
  EmptyState,
  EmptyStateDescription,
  EmptyStateHeader,
  EmptyStateIcon,
  EmptyStateTitle,
  ResourceListBody,
  ResourceListContent,
  ResourceListItem,
  type ResourceListView,
  Skeleton,
} from "@unkey/ui";
import { cn } from "cn";
import type { Route } from "next";
import Link from "next/link";
import { type CSSProperties, type ComponentProps, Fragment, type ReactNode } from "react";

type ListColumnSkeleton = "title" | "text" | "none";

export type ListColumn<Id extends string> = {
  id: Id;
  header: ReactNode;
  width: string;
  skeleton: ListColumnSkeleton;
};

export type ListTable<Id extends string> = {
  columns: readonly ListColumn<Id>[];
  minWidth: number;
};

export function listTable<const Id extends string>(table: ListTable<Id>): ListTable<Id> {
  return table;
}

type ShownContent<Row> = Exclude<ListContent<Row>, { status: "noMatch" }>;

type ListViewsProps<Row, Id extends string> = {
  view: ResourceListView;
  content: ListContent<Row>;
  noun: string;
  table: ListTable<Id>;
  rowKey: (row: Row) => string;
  renderCard: (row: Row) => ReactNode;
  renderRow: (row: Row, table: ListTable<Id>) => ReactNode;
  cardSkeleton: ReactNode;
};

type GridProps<Row> = Pick<
  ListViewsProps<Row, string>,
  "noun" | "rowKey" | "renderCard" | "cardSkeleton"
> & {
  content: ShownContent<Row>;
};

type TableProps<Row, Id extends string> = Pick<
  ListViewsProps<Row, Id>,
  "noun" | "rowKey" | "renderRow" | "table"
> & {
  content: ShownContent<Row>;
};

const SKELETON_KEYS = ["skeleton-1", "skeleton-2", "skeleton-3"];

const SKELETON_CELLS: Record<ListColumnSkeleton, ReactNode> = {
  title: <Skeleton className="h-3 w-28" />,
  text: <Skeleton className="h-3 w-16" />,
  none: <span />,
};

export function ListViews<Row, Id extends string>({
  view,
  content,
  renderCard,
  renderRow,
  cardSkeleton,
  table,
  ...shared
}: ListViewsProps<Row, Id>) {
  return match(content)
    .with({ status: "noMatch" }, ({ search }) => <ListNoMatch noun={shared.noun} search={search} />)
    .with({ status: "loading" }, { status: "ready" }, (shown) =>
      match(view)
        .with("grid", () => (
          <ListGrid
            {...shared}
            content={shown}
            renderCard={renderCard}
            cardSkeleton={cardSkeleton}
          />
        ))
        .with("list", () => (
          <ListTableView {...shared} content={shown} renderRow={renderRow} table={table} />
        ))
        .exhaustive(),
    )
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

function ListGrid<Row>({ content, noun, rowKey, renderCard, cardSkeleton }: GridProps<Row>) {
  return match(content)
    .with({ status: "loading" }, () => (
      <div aria-busy="true">
        <output className="sr-only">Loading {noun}...</output>
        <div aria-hidden="true" className="grid grid-cols-1 gap-4 md:grid-cols-2 xl:grid-cols-3">
          {SKELETON_KEYS.map((key) => (
            <Fragment key={key}>{cardSkeleton}</Fragment>
          ))}
        </div>
      </div>
    ))
    .with({ status: "ready" }, ({ rows, footer }) => (
      <>
        <div className="grid grid-cols-1 gap-4 md:grid-cols-2 xl:grid-cols-3">
          {rows.map((row) => (
            <Fragment key={rowKey(row)}>{renderCard(row)}</Fragment>
          ))}
        </div>
        <ListPager state={footer} noun={noun} variant="bare" />
      </>
    ))
    .exhaustive();
}

function ListTableView<Row, Id extends string>({
  content,
  noun,
  table,
  rowKey,
  renderRow,
}: TableProps<Row, Id>) {
  const { columns, minWidth } = table;
  const style: CSSProperties & Record<"--list-columns", string> = {
    "--list-columns": columns.map((column) => column.width).join(" "),
  };
  const { announcement, body, pager } = match(content)
    .with({ status: "loading" }, () => ({
      announcement: <output className="sr-only">Loading {noun}...</output>,
      body: (
        <ResourceListBody aria-hidden="true">
          {SKELETON_KEYS.map((key) => (
            <ResourceListItem
              key={key}
              className="grid h-12 grid-cols-(--list-columns) items-center gap-4 px-4"
            >
              {columns.map((column) => (
                <Fragment key={column.id}>{SKELETON_CELLS[column.skeleton]}</Fragment>
              ))}
            </ResourceListItem>
          ))}
        </ResourceListBody>
      ),
      pager: null,
    }))
    .with({ status: "ready" }, ({ rows, footer }) => ({
      announcement: null,
      body: (
        <ResourceListBody>
          {rows.map((row) => (
            <Fragment key={rowKey(row)}>{renderRow(row, table)}</Fragment>
          ))}
        </ResourceListBody>
      ),
      pager: <ListPager state={footer} noun={noun} />,
    }))
    .exhaustive();

  return (
    <ResourceListContent aria-busy={content.status === "loading"} style={style}>
      {announcement}
      <div className="overflow-x-auto">
        <div style={{ minWidth }}>
          <div
            aria-hidden="true"
            className="grid grid-cols-(--list-columns) items-center gap-4 border-b bg-table-header px-4 py-[7px] text-xs font-medium text-gray-12"
          >
            {columns.map((column) => (
              <span key={column.id}>{column.header}</span>
            ))}
          </div>
          {body}
        </div>
      </div>
      {pager}
    </ResourceListContent>
  );
}

export function ListTableRow<Id extends string>({
  table,
  cells,
  href,
  label,
  className,
  ...props
}: Omit<ComponentProps<"li">, "children"> & {
  table: ListTable<Id>;
  cells: Record<Id, ReactNode>;
  href: Route;
  label: string;
}) {
  return (
    <ResourceListItem
      className={cn(
        "grid h-12 grid-cols-(--list-columns) items-center gap-4 px-4 text-xs text-gray-12 transition-colors hover:bg-grayA-2 has-[>a:focus-visible]:bg-grayA-2",
        className,
      )}
      {...props}
    >
      <Link href={href} className="absolute inset-0 z-0 outline-hidden" aria-label={label} />
      {table.columns.map((column) => (
        <Fragment key={column.id}>{cells[column.id]}</Fragment>
      ))}
    </ResourceListItem>
  );
}
