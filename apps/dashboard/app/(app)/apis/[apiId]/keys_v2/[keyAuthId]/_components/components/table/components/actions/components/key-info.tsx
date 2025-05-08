import { RatelimitOverviewTooltip } from "@/app/(app)/ratelimits/[namespaceId]/_overview/components/table/components/ratelimit-overview-tooltip";
import type { KeyDetails } from "@/lib/trpc/routers/api/keys/query-api-keys/schema";
import { Key2 } from "@unkey/icons";

export const KeyInfo = ({ keyDetails }: { keyDetails: KeyDetails }) => {
  return (
    <div className="flex gap-5 items-center bg-white dark:bg-black border border-grayA-5 rounded-xl py-5 pl-[18px] pr-[26px]">
      <div className="bg-grayA-5 text-gray-12 size-5 flex items-center justify-center rounded ">
        <Key2 size="sm-regular" />
      </div>
      <div className="flex flex-col gap-1">
        <div className="text-accent-12 text-xs font-mono">{keyDetails.id}</div>
        <RatelimitOverviewTooltip
          content={keyDetails.name}
          position={{ side: "bottom", align: "center" }}
          asChild
          disabled={!keyDetails.name}
        >
          <div className="text-accent-9 text-xs max-w-[160px] truncate">
            {keyDetails.name ?? "Unnamed Key"}
          </div>
        </RatelimitOverviewTooltip>
      </div>
    </div>
  );
};
