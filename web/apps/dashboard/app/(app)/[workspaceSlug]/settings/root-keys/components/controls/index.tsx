import { ResourceSearchInput } from "@/components/resource-search-input";
import { ControlsEndSlot } from "@unkey/ui";

export function RootKeysListControls() {
  return (
    <div className="@container/controls flex w-full items-center gap-2">
      <ResourceSearchInput
        queryKey="name"
        label="Search Root Keys"
        placeholder="Search Root Keys by name..."
      />
      <ControlsEndSlot className="ml-auto" />
    </div>
  );
}
