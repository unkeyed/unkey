import { ResourceSearchInput } from "@/components/resource-search-input";

export function RootKeysListControls() {
  return (
    <div className="flex w-full items-center">
      <ResourceSearchInput
        queryKey="name"
        label="Search root keys"
        placeholder="Search root keys by name..."
      />
    </div>
  );
}
