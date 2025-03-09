import { LogsDateTime } from "./components/logs-datetime";
import { LogsQueries } from "./components/logs-queries";
import { LogsRefresh } from "./components/logs-refresh";
import { LogsSearch } from "./components/logs-search";

type RatelimitListControlsProps = {
  setNamespaces: (namespaces: { id: string; name: string }[]) => void;
  initialNamespaces: { id: string; name: string }[];
};

export function RatelimitListControls({
  setNamespaces,
  initialNamespaces,
}: RatelimitListControlsProps) {
  return (
    <div className="flex flex-col border-b border-gray-4">
      <div className="flex items-center justify-between w-full px-3 py-1">
        <div className="flex gap-2">
          <div className="flex items-center gap-2">
            <LogsSearch setNamespaces={setNamespaces} initialNamespaces={initialNamespaces} />
          </div>
          <div className="flex items-center gap-2">
            <LogsDateTime />
          </div>
          <div className="flex items-center gap-2">
            <LogsQueries storageName="ratelimitSavedFilters" />
          </div>
        </div>
        <div className="flex gap-2">
          <LogsRefresh />
        </div>
      </div>
    </div>
  );
}
