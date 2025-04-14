import { LogsFilters } from "./components/logs-filters";
import { LogsSearch } from "./components/logs-search";

export function KeysListControls({ keyspaceId }: { keyspaceId: string }) {
  return (
    <div className="flex flex-col border-b border-gray-4 ">
      <div className="px-3 py-1 w-full justify-between flex items-center">
        <div className="flex gap-2">
          <div className="flex gap-2 items-center">
            <LogsSearch keyspaceId={keyspaceId} />
          </div>
          <div className="flex gap-2 items-center">
            <LogsFilters />
          </div>
        </div>
      </div>
    </div>
  );
}
