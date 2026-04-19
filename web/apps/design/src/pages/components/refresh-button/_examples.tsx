import { RefreshButton } from "@unkey/ui";
import { useState } from "react";
import { Preview } from "../../../components/Preview";

export function BasicExample() {
  const [lastRefresh, setLastRefresh] = useState<string | null>(null);

  return (
    <Preview>
      <div className="flex flex-col items-center gap-3">
        <RefreshButton
          onRefresh={() => setLastRefresh(new Date().toLocaleTimeString())}
          isEnabled
        />
        <span className="text-xs text-gray-10">
          {lastRefresh ? `Last refresh: ${lastRefresh}` : "Not refreshed yet"}
        </span>
      </div>
    </Preview>
  );
}

export function DisabledExample() {
  return (
    <Preview>
      <RefreshButton onRefresh={() => {}} isEnabled={false} />
    </Preview>
  );
}

export function LiveModeExample() {
  const [isLive, setIsLive] = useState(true);
  const [lastRefresh, setLastRefresh] = useState<string | null>(null);

  return (
    <Preview>
      <div className="flex flex-col items-center gap-3">
        <RefreshButton
          onRefresh={() => setLastRefresh(new Date().toLocaleTimeString())}
          isEnabled
          isLive={isLive}
          toggleLive={setIsLive}
        />
        <span className="text-xs text-gray-10">
          Live mode is {isLive ? "on" : "off"}
          {lastRefresh ? ` — last refresh ${lastRefresh}` : ""}
        </span>
      </div>
    </Preview>
  );
}
