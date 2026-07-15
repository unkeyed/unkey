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
    <div className="flex min-h-10 w-full items-center gap-2">
      <div className="w-full md:w-[calc((100%-1.25rem)/2)] xl:w-[calc((100%-2.5rem)/3)]">
        <LogsSearch {...props} />
      </div>
      <LogsDateTime />
    </div>
  );
}
