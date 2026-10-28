"use client";

import { IconGridOutline18, IconSquareBulletListOutline18 } from "@unkey/icons";
import { Tabs, TabsList, TabsTrigger } from "../tabs";

export const RESOURCE_LIST_VIEWS = ["grid", "list"] as const;
export type ResourceListView = (typeof RESOURCE_LIST_VIEWS)[number];

export function parseResourceListView(value: string | null): ResourceListView | undefined {
  return RESOURCE_LIST_VIEWS.find((view) => view === value);
}

const VIEW_OPTIONS: Record<ResourceListView, { label: string; Icon: typeof IconGridOutline18 }> = {
  grid: { label: "Grid view", Icon: IconGridOutline18 },
  list: { label: "List view", Icon: IconSquareBulletListOutline18 },
};

export function ResourceListViewToggle({
  value,
  onValueChange,
  className,
}: {
  value: ResourceListView;
  onValueChange: (view: ResourceListView) => void;
  className?: string;
}) {
  return (
    <Tabs
      value={value}
      onValueChange={(next) => {
        const view = parseResourceListView(next);
        if (view) {
          onValueChange(view);
        }
      }}
      className={className}
    >
      <TabsList aria-label="View" className="h-8 gap-0.5 border bg-raised p-0.5">
        {RESOURCE_LIST_VIEWS.map((view) => {
          const { label, Icon } = VIEW_OPTIONS[view];
          return (
            <TabsTrigger
              key={view}
              value={view}
              aria-label={label}
              className="h-full w-7 px-0 py-0 text-gray-11 hover:text-gray-12 data-active:bg-grayA-3 data-active:text-gray-12 data-active:shadow-none"
            >
              <Icon className="size-3.5" />
            </TabsTrigger>
          );
        })}
      </TabsList>
    </Tabs>
  );
}
