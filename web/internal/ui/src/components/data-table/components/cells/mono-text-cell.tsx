import { cn } from "../../../../lib/utils";

export interface MonoTextCellProps {
  value: string;
  className?: string;
}

/**
 * Monospace text cell with truncation for table columns
 */
export function MonoTextCell({ value, className }: MonoTextCellProps) {
  return (
    <div
      className={cn(
        "font-mono text-xs overflow-hidden text-ellipsis whitespace-nowrap truncate min-w-0",
        className,
      )}
    >
      {value}
    </div>
  );
}
