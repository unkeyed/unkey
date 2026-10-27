"use client";
import { ResourceSearchInput } from "@/components/resource-search-input";
import { ResourceListHeader, type ResourceListView, ResourceListViewToggle } from "@unkey/ui";

type Props = {
  view: ResourceListView;
  onViewChange: (view: ResourceListView) => void;
};

export function AppsListControls({ view, onViewChange }: Props) {
  return (
    <ResourceListHeader>
      <ResourceSearchInput
        label="Search apps"
        placeholder="Search by name, repo, image or domain"
      />
      <ResourceListViewToggle value={view} onValueChange={onViewChange} />
    </ResourceListHeader>
  );
}
