import { ListSearchInput } from "@/components/list-search-input";
import { useNamespaceListFilters } from "../hooks/use-namespace-list-filters";
import { NamespaceListDateTime } from "./components/namespace-list-datetime";

export function NamespaceListControls() {
  return (
    <div className="flex min-h-10 w-full items-center gap-2">
      <div className="w-full md:w-[calc((100%-1.25rem)/2)] xl:w-[calc((100%-2.5rem)/3)]">
        <ListSearchInput
          useFiltersHook={useNamespaceListFilters}
          placeholder="Search namespaces..."
        />
      </div>
      <NamespaceListDateTime />
    </div>
  );
}
