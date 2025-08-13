"use client";

import type { ViewMode } from "../../hooks/use-view-mode";
import { ViewModeToggle } from "./view-mode-toggle";

interface ChartHeaderProps {
  title: string;
  viewMode: ViewMode;
  onViewModeChange: (mode: ViewMode) => void;
  showViewModeToggle?: boolean;
  className?: string;
}

export function ChartHeader({
  title,
  viewMode,
  onViewModeChange,
  showViewModeToggle = true,
  className = "",
}: ChartHeaderProps) {
  return (
    <div className={`flex items-center justify-between p-4 border-b border-gray-4 ${className}`}>
      <h3 className="text-sm font-medium text-gray-12">{title}</h3>
      {showViewModeToggle && (
        <ViewModeToggle
          viewMode={viewMode}
          onViewModeChange={onViewModeChange}
          className="w-[200px]"
        />
      )}
    </div>
  );
}
