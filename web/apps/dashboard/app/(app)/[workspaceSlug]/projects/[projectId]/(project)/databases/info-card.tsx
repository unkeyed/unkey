import { cn } from "cn";
import type { ComponentType, ReactNode } from "react";

type Icon = ComponentType<{ className?: string }>;

export function InfoCard({ children }: { children: ReactNode }) {
  return (
    <div className="flex flex-col divide-y divide-grayA-4 overflow-hidden rounded-lg border border-grayA-4 bg-raised">
      {children}
    </div>
  );
}

export function InfoRow({
  icon: Icon,
  label,
  children,
}: { icon: Icon; label: string; children: ReactNode }) {
  return (
    <dl className="flex min-h-11 items-center justify-between gap-4 px-4 py-2.5 text-sm">
      <dt className="flex shrink-0 items-center gap-2 font-medium text-gray-11">
        <Icon className="size-3.5 text-gray-10" aria-hidden="true" />
        {label}
      </dt>
      <dd className="flex min-w-0 items-center justify-end gap-2 text-right font-medium text-gray-12">
        {children}
      </dd>
    </dl>
  );
}

const CELL_COLUMNS = { 2: "grid-cols-2", 4: "grid-cols-4" } as const;

export function InfoCells({
  columns,
  children,
}: { columns: keyof typeof CELL_COLUMNS; children: ReactNode }) {
  return <dl className={cn("grid divide-x divide-grayA-4", CELL_COLUMNS[columns])}>{children}</dl>;
}

export function InfoCell({
  icon: Icon,
  label,
  value,
}: { icon: Icon; label: string; value: ReactNode }) {
  return (
    <div className="flex min-w-0 flex-col gap-1.5 px-4 py-3">
      <dt className="flex items-center gap-2 text-sm font-medium text-gray-11">
        <Icon className="size-3.5 text-gray-10" aria-hidden="true" />
        {label}
      </dt>
      <dd className="text-lg font-medium leading-6 tabular-nums text-gray-12">{value}</dd>
    </div>
  );
}
