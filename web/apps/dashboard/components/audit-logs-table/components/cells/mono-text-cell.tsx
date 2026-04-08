import { cn } from "@/lib/utils";

type MonoTextCellProps = {
  value: string;
  className?: string;
};

export const MonoTextCell = ({ value, className }: MonoTextCellProps) => {
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
};
