import { LogsDateTime } from "./components/logs-datetime";
import { LogsFilters } from "./components/logs-filters";
import { LogsRefresh } from "./components/logs-refresh";
import { LogsSearch } from "./components/logs-search";

export function KeysDetailsLogsControls({
  keyspaceId,
  keyId,
}: {
  keyId: string;
  keyspaceId: string;
}) {
  return (
    <div className="flex flex-col border-b border-gray-4 w-full">
      <div className="px-3 py-1 w-full justify-between flex items-center">
        <div className="flex gap-2 w-full">
          <div className="flex flex-1 gap-2 items-center">
            <LogsSearch keyspaceId={keyspaceId} keyId={keyId} />
          </div>
          <div className="flex gap-2 md:w-full max-md:justify-end">
            <div className="flex gap-2 items-center">
              <LogsFilters />
            </div>
            <div className="flex gap-2 items-center">
              <LogsDateTime />
            </div>
          </div>
        </div>

        <div className="flex gap-2 max-md:hidden">
          <LogsRefresh />
        </div>
      </div>
    </div>
  );
}
