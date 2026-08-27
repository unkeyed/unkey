import { ResourceSearchInput } from "@/components/resource-search-input";

export function RootKeysListControls() {
  return (
    <div className="flex w-full items-center">
      <ResourceSearchInput
        queryKey="name"
        label="Search Root Keys"
        placeholder="Search Root Keys by name..."
      />
    </div>
  );
}
