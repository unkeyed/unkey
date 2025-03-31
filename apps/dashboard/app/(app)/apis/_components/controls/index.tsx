import type { ApiOverview } from "@/lib/trpc/routers/api/overview/query-overview/schemas";
import { LogsDateTime } from "./components/logs-datetime";
import { LogsRefresh } from "./components/logs-refresh";
import { LogsSearch } from "./components/logs-search";

type Props = {
  apiList: ApiOverview[];
  onApiListChange: (apiList: ApiOverview[]) => void;
  onSearch: (value: boolean) => void;
};

export function ApiListControls(props: Props) {
  return (
    <div className="flex flex-col border-b border-gray-4">
      <div className="px-3 py-1 w-full justify-between flex items-center">
        <div className="flex gap-2">
          <div className="flex gap-2 items-center">
            <LogsSearch {...props} />
          </div>
          <div className="flex gap-2 items-center">
            <LogsDateTime />
          </div>
        </div>
        <div className="flex gap-2">
          <LogsRefresh />
        </div>
      </div>
    </div>
  );
}
