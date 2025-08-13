"use client";

import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs";
import type { ViewMode } from "../../hooks/use-view-mode";

interface ViewModeToggleProps {
  viewMode: ViewMode;
  onViewModeChange: (mode: ViewMode) => void;
  className?: string;
}

export function ViewModeToggle({ viewMode, onViewModeChange, className }: ViewModeToggleProps) {
  return (
    <Tabs
      value={viewMode}
      onValueChange={(value) => onViewModeChange(value as ViewMode)}
      className={className}
    >
      <TabsList className="grid w-full grid-cols-2">
        <TabsTrigger value="verifications" className="text-xs">
          Verifications
        </TabsTrigger>
        <TabsTrigger value="credits" className="text-xs">
          Credits Spent
        </TabsTrigger>
      </TabsList>
    </Tabs>
  );
}
