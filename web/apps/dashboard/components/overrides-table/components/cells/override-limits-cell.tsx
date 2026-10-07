import { formatNumber } from "@/lib/fmt";
import { formatMs } from "@/lib/ms";
import { Badge } from "@unkey/ui";

type OverrideLimitsCellProps = {
  limit: number;
  duration: number;
};

export const OverrideLimitsCell = ({ limit, duration }: OverrideLimitsCellProps) => {
  return (
    <div className="flex justify-start">
      <Badge className="px-2 rounded-md font-mono truncate uppercase bg-gray-4 text-gray-11 group-hover:bg-gray-5">
        {formatNumber(limit)}/{formatMs(duration)}
      </Badge>
    </div>
  );
};
