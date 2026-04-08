import { cn } from "../../../../lib/utils";

export interface RegionCellProps {
  region: string;
  className?: string;
}

export function RegionCell({ region, className }: RegionCellProps) {
  return (
    <div className={cn("flex items-center font-mono", className)}>
      <div className="w-full whitespace-nowrap" title={region}>
        {region}
      </div>
    </div>
  );
}
