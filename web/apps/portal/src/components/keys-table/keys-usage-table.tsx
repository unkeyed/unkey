import { ArrowDown, ArrowUp, ArrowUpDown, ChevronLeft, ChevronRight, KeyRound } from "lucide-react";
import { formatCount } from "~/components/analytics/format";
import type { KeyUsage } from "~/components/analytics/key-usage";
import { OUTCOME_LABELS, nonZeroOutcomes } from "~/components/analytics/outcomes";
import { Badge } from "~/components/ui/badge";
import { Button } from "~/components/ui/button";
import {
  Empty,
  EmptyContent,
  EmptyDescription,
  EmptyHeader,
  EmptyMedia,
  EmptyTitle,
} from "~/components/ui/empty";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "~/components/ui/table";
import { PAGE_SIZES, useTablePaging } from "~/hooks/use-table-paging";
import { cn } from "~/lib/utils";
import { type KeyRow, ariaSort } from "./key-row";
import { type KeyStatus, keyStatus } from "./key-status";
import type { Key } from "./schema/keys.schema";

const SKELETON_ROWS = ["first", "second", "third"];

const FIRST_CELL = "pl-5";
const NAME_CELL = "pl-5 sm:pl-3";
const LAST_CELL = "pr-5 sm:pr-6";

type Props = {
  rows: KeyRow[];
  isLoading: boolean;
  showUsage: boolean;
  /** Set when a filter is active, so an empty result can offer a reset. */
  onClearFilters?: () => void;
  /** Omitted for a session with no action it may take, which leaves the cell empty. */
  renderActions?: (key: Key) => React.ReactNode;
};

export function KeysUsageTable({
  rows,
  isLoading,
  showUsage,
  onClearFilters,
  renderActions,
}: Props) {
  const paging = useTablePaging(rows, showUsage);

  return (
    <section className="overflow-hidden rounded-b-lg" aria-label="API keys">
      {!isLoading && rows.length === 0 ? (
        <NoKeys onClearFilters={onClearFilters} />
      ) : (
        <Table className="table-fixed">
          <TableHeader>
            <TableRow className="hover:bg-transparent">
              <Head className={cn("hidden w-40 sm:table-cell", FIRST_CELL)}>Key</Head>
              <Head className={NAME_CELL}>Name</Head>
              <Head className="hidden w-28 sm:table-cell">Status</Head>
              {showUsage && (
                <>
                  <Head className="w-20 sm:w-24" aria-sort={ariaSort(paging.sort, "valid")}>
                    <SortButton
                      label="Valid"
                      active={paging.sort.key === "valid"}
                      desc={paging.sort.desc}
                      onClick={() => paging.toggleSort("valid")}
                    />
                  </Head>
                  <Head className="w-20 sm:w-24" aria-sort={ariaSort(paging.sort, "error")}>
                    <SortButton
                      label="Invalid"
                      active={paging.sort.key === "error"}
                      desc={paging.sort.desc}
                      onClick={() => paging.toggleSort("error")}
                    />
                  </Head>
                </>
              )}
              <Head className={cn("w-16", LAST_CELL)}>
                <span className="sr-only">Actions</span>
              </Head>
            </TableRow>
          </TableHeader>
          <TableBody>
            {isLoading ? (
              <SkeletonRows showUsage={showUsage} />
            ) : (
              paging.pageRows.map((row) => (
                <KeyTableRow
                  key={row.key.id}
                  row={row}
                  showUsage={showUsage}
                  renderActions={renderActions}
                />
              ))
            )}
          </TableBody>
        </Table>
      )}
      {!isLoading && paging.pageCount > 1 && (
        <TablePagination
          pageStart={paging.pageStart}
          pageSize={paging.pageSize}
          pageCount={paging.pageCount}
          currentPage={paging.currentPage}
          total={paging.total}
          setPageSize={paging.setPageSize}
          setPage={paging.setPage}
        />
      )}
    </section>
  );
}

function SkeletonRows({ showUsage }: { showUsage: boolean }) {
  return (
    <>
      {SKELETON_ROWS.map((id) => (
        <TableRow key={id} className="h-12 hover:bg-transparent" aria-busy="true">
          <TableCell className={cn("hidden sm:table-cell", FIRST_CELL)}>
            <SkeletonCell className="w-28" />
          </TableCell>
          <TableCell className={NAME_CELL}>
            <SkeletonCell className="w-32" />
          </TableCell>
          <TableCell className="hidden sm:table-cell">
            <SkeletonCell className="w-16" />
          </TableCell>
          {showUsage && (
            <>
              <TableCell>
                <SkeletonCell className="w-12" />
              </TableCell>
              <TableCell>
                <SkeletonCell className="w-12" />
              </TableCell>
            </>
          )}
          <TableCell className={LAST_CELL}>
            <SkeletonCell className="h-6 w-6" />
          </TableCell>
        </TableRow>
      ))}
    </>
  );
}

function KeyTableRow({
  row: { key, usage, pending },
  showUsage,
  renderActions,
}: {
  row: KeyRow;
  showUsage: boolean;
  renderActions?: (key: Key) => React.ReactNode;
}) {
  const status = keyStatus(key);
  return (
    <TableRow>
      <TableCell className={cn("hidden sm:table-cell", FIRST_CELL)}>
        <span className="font-mono text-gray-11 text-xs">{key.start.padEnd(16, "•")}</span>
      </TableCell>
      <TableCell className={NAME_CELL}>
        <span
          className={cn(
            "block truncate font-medium text-gray-12",
            key.name === null && "font-mono text-xs",
          )}
          title={key.name ?? key.id}
        >
          {key.name ?? key.id}
        </span>
      </TableCell>
      <TableCell className="hidden sm:table-cell">
        <StatusBadge status={status} />
      </TableCell>
      {showUsage && (
        <>
          <TableCell>
            <Count value={usage ? usage.valid : null} pending={pending} />
          </TableCell>
          <TableCell title={usage && usage.error > 0 ? outcomeSummary(usage) : undefined}>
            <Count value={usage ? usage.error : null} pending={pending} />
          </TableCell>
        </>
      )}
      <TableCell className={cn("py-1 text-right", LAST_CELL)}>{renderActions?.(key)}</TableCell>
    </TableRow>
  );
}

function TablePagination({
  pageStart,
  pageSize,
  pageCount,
  currentPage,
  total,
  setPageSize,
  setPage,
}: {
  pageStart: number;
  pageSize: number;
  pageCount: number;
  currentPage: number;
  total: number;
  setPageSize: (size: number) => void;
  setPage: (page: number) => void;
}) {
  return (
    <footer className="flex flex-wrap items-center justify-between gap-3 border-primary/10 border-t px-5 py-3 text-gray-11 text-xs">
      <span aria-live="polite">
        {pageStart + 1}–{Math.min(pageStart + pageSize, total)} of {total}{" "}
        {total === 1 ? "key" : "keys"}
      </span>
      <div className="flex items-center gap-3">
        <label className="flex items-center gap-2">
          <span className="hidden sm:inline">Rows per page</span>
          <select
            aria-label="Rows per page"
            value={pageSize}
            onChange={(event) => setPageSize(Number(event.target.value))}
            className="h-7 rounded-md border border-primary/10 bg-background px-1.5 text-gray-12 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-gray-12"
          >
            {PAGE_SIZES.map((size) => (
              <option key={size} value={size}>
                {size}
              </option>
            ))}
          </select>
        </label>
        <span>
          Page {currentPage + 1} of {pageCount}
        </span>
        <div className="flex gap-1">
          <Button
            variant="outline"
            size="icon"
            aria-label="Previous page"
            disabled={currentPage === 0}
            onClick={() => setPage(currentPage - 1)}
          >
            <ChevronLeft />
          </Button>
          <Button
            variant="outline"
            size="icon"
            aria-label="Next page"
            disabled={currentPage === pageCount - 1}
            onClick={() => setPage(currentPage + 1)}
          >
            <ChevronRight />
          </Button>
        </div>
      </div>
    </footer>
  );
}

function NoKeys({ onClearFilters }: { onClearFilters?: () => void }) {
  return (
    <Empty className="py-12">
      <EmptyHeader>
        <EmptyMedia variant="icon">
          <KeyRound />
        </EmptyMedia>
        <EmptyTitle>{onClearFilters ? "No keys match your filters" : "No keys yet"}</EmptyTitle>
        <EmptyDescription>
          {onClearFilters
            ? "Try a different key, outcome, status or time range."
            : "Your API keys will show up here once you have one."}
        </EmptyDescription>
      </EmptyHeader>
      {onClearFilters && (
        <EmptyContent>
          <Button variant="ghost" onClick={onClearFilters}>
            Clear filters
          </Button>
        </EmptyContent>
      )}
    </Empty>
  );
}

function SortButton({
  label,
  active,
  desc,
  onClick,
}: {
  label: string;
  active: boolean;
  desc: boolean;
  onClick: () => void;
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      className="group inline-flex items-center gap-1 rounded-sm transition-colors hover:text-gray-12 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-gray-12 focus-visible:ring-offset-1"
    >
      {label}
      {active ? (
        desc ? (
          <ArrowDown className="size-3 text-gray-12" />
        ) : (
          <ArrowUp className="size-3 text-gray-12" />
        )
      ) : (
        <ArrowUpDown className="size-3 text-gray-9 opacity-0 transition-opacity group-hover:opacity-100 group-focus-visible:opacity-100" />
      )}
    </button>
  );
}

function Count({ value, pending }: { value: number | null; pending: boolean }) {
  if (pending) {
    return <SkeletonCell className="w-10" />;
  }
  return value === null ? (
    <span className="text-gray-9">—</span>
  ) : (
    <span className="tabular-nums">{formatCount(value)}</span>
  );
}

function StatusBadge({ status }: { status: KeyStatus }) {
  if (status === "enabled") {
    return <Badge variant="success">Enabled</Badge>;
  }
  if (status === "expired") {
    return <Badge variant="error">Expired</Badge>;
  }
  return <Badge variant="secondary">Disabled</Badge>;
}

function SkeletonCell({ className }: { className?: string }) {
  return <div className={cn("h-4 rounded bg-gray-3 motion-safe:animate-pulse", className)} />;
}

function Head({ children, className, ...props }: React.ThHTMLAttributes<HTMLTableCellElement>) {
  return (
    <TableHead className={cn("text-gray-11", className)} {...props}>
      {children}
    </TableHead>
  );
}

function outcomeSummary(usage: KeyUsage): string {
  const parts = nonZeroOutcomes(usage).map(
    ({ kind, count }) => `${formatCount(count)} ${OUTCOME_LABELS[kind].toLowerCase()}`,
  );
  if (usage.other > 0) {
    parts.push(`${formatCount(usage.other)} other`);
  }
  return parts.join(" · ");
}
