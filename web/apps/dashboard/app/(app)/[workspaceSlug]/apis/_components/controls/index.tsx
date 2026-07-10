import { ControlsLeft } from "@/components/logs/controls-container";
import type { ApiOverview } from "@/lib/trpc/routers/api/overview/query-overview/schemas";
import { LogsDateTime } from "./components/logs-datetime";
import { LogsSearch } from "./components/logs-search";

type Props = {
  apiList: ApiOverview[];
  onApiListChange: (apiList: ApiOverview[]) => void;
  onSearch: (value: boolean) => void;
};

export function ApiListControls(props: Props) {
  return (
    <div className="flex min-h-10 w-full items-center justify-between gap-2">
      <ControlsLeft>
        <LogsSearch {...props} />
        <LogsDateTime />
      </ControlsLeft>
    </div>
  );
}
