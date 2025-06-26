"use client";
import { RenderComponentWithSnippet } from "@/app/components/render";
import { Button, RefreshButton } from "@unkey/ui";
import { useState } from "react";

export const Default = () => {
  const [refreshCount, setRefreshCount] = useState(0);

  const handleRefresh = () => {
    setRefreshCount((prev) => prev + 1);
  };

  return (
    <RenderComponentWithSnippet>
      <div className="flex flex-col gap-6">
        <div>
          <h4 className="text-sm font-medium mb-2">Basic Refresh Button</h4>
          <div className="flex items-center gap-4">
            <RefreshButton onRefresh={handleRefresh} isEnabled={true} />
            <span className="text-sm text-gray-600">Refresh count: {refreshCount}</span>
          </div>
        </div>

        <div>
          <h4 className="text-sm font-medium mb-2">With Custom Styling</h4>
          <div className="flex items-center gap-4">
            <RefreshButton onRefresh={handleRefresh} isEnabled={true} />
            <span className="text-xs text-gray-500">Try clicking or pressing ⌥+⇧+R</span>
          </div>
        </div>
      </div>
    </RenderComponentWithSnippet>
  );
};

export const WithLiveMode = () => {
  const [refreshCount, setRefreshCount] = useState(0);
  const [isLive, setIsLive] = useState(true);
  const [liveData, setLiveData] = useState(`Live data: ${Date.now()}`);

  const handleRefresh = () => {
    setRefreshCount((prev) => prev + 1);
    setLiveData(`Live data: ${Date.now()}`);
  };

  return (
    <RenderComponentWithSnippet>
      <div className="flex flex-col gap-6">
        <div>
          <h4 className="text-sm font-medium mb-2">With Live Mode Integration</h4>
          <div className="flex items-center gap-4">
            <RefreshButton
              onRefresh={handleRefresh}
              isEnabled={true}
              isLive={isLive}
              toggleLive={setIsLive}
            />
            <div className="flex flex-col gap-1">
              <span className="text-sm text-gray-600">Refresh count: {refreshCount}</span>
              <span className="text-sm text-gray-600">{liveData}</span>
              <span className="text-xs text-gray-500">Live mode: {isLive ? "ON" : "OFF"}</span>
            </div>
          </div>
        </div>

        <div>
          <h4 className="text-sm font-medium mb-2">Live Mode Toggle</h4>
          <div className="flex items-center gap-4">
            <Button
              variant="outline"
              onClick={() => setIsLive(!isLive)}
              className="px-3 py-1 text-sm border rounded hover:bg-gray-50"
            >
              Toggle Live Mode
            </Button>
            <span className="text-xs text-gray-500">
              Live mode will be temporarily disabled during refresh
            </span>
          </div>
        </div>
      </div>
    </RenderComponentWithSnippet>
  );
};

export const DisabledState = () => {
  const [timeFilter, setTimeFilter] = useState("1h");
  const [refreshCount, setRefreshCount] = useState(0);

  const handleRefresh = () => {
    setRefreshCount((prev) => prev + 1);
  };

  // Disable refresh when "all" time filter is selected
  const isEnabled = timeFilter !== "all";

  return (
    <RenderComponentWithSnippet>
      <div className="flex flex-col gap-6">
        <div>
          <h4 className="text-sm font-medium mb-2">Conditional Enablement</h4>
          <div className="flex items-center gap-4">
            <select
              value={timeFilter}
              onChange={(e) => setTimeFilter(e.target.value)}
              className="px-3 py-1 text-sm border rounded"
            >
              <option value="1h">Last 1 hour</option>
              <option value="24h">Last 24 hours</option>
              <option value="7d">Last 7 days</option>
              <option value="all">All time</option>
            </select>
            <RefreshButton onRefresh={handleRefresh} isEnabled={isEnabled} />
            <span className="text-sm text-gray-600">Refresh count: {refreshCount}</span>
          </div>
        </div>

        <div>
          <h4 className="text-sm font-medium mb-2">Disabled State</h4>
          <div className="flex items-center gap-4">
            <RefreshButton onRefresh={handleRefresh} isEnabled={false} />
            <span className="text-xs text-gray-500">
              Hover over the disabled button to see the tooltip
            </span>
          </div>
        </div>

        <div>
          <h4 className="text-sm font-medium mb-2">State Comparison</h4>
          <div className="flex items-center gap-4">
            <div className="flex flex-col gap-2">
              <span className="text-xs font-medium">Enabled:</span>
              <RefreshButton onRefresh={handleRefresh} isEnabled={true} />
            </div>
            <div className="flex flex-col gap-2">
              <span className="text-xs font-medium">Disabled:</span>
              <RefreshButton onRefresh={handleRefresh} isEnabled={false} />
            </div>
          </div>
        </div>
      </div>
    </RenderComponentWithSnippet>
  );
};
