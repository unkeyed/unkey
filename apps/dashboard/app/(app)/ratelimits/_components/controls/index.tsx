import { LogsDateTime } from "./components/logs-datetime";
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
      <div className="px-3 py-1 w-full justify-between flex items-center">
        <div className="flex gap-2">
          <div className="flex gap-2 items-center">
            <LogsSearch setNamespaces={setNamespaces} initialNamespaces={initialNamespaces} />
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
