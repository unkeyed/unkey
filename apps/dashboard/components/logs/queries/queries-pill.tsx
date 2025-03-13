import { cn } from "@/lib/utils";
import { parseValue } from "./utils";
type QueriesPillType = {
  value: string;
  className?: string;
};
export const QueriesPill = ({ value, className }: QueriesPillType) => {
  const { color, phrase } = parseValue(value);

  return (
    <div
      className={cn(
        "h-6 bg-gray-3 inline-flex justify-start items-center py-1.5 px-2  rounded-md gap-2 ",
        className,
      )}
    >
      {color && <div className={cn("w-2 h-2 rounded-[2px]", color)} />}
      <span className="font-mono text-xs font-medium truncate text-gray-12">{phrase}</span>
    </div>
  );
};
