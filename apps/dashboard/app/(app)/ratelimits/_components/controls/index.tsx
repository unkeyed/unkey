import { ListSearchInput } from "@/components/list-search-input";
import {
  ControlsContainer,
  ControlsLeft,
  ControlsRight,
} from "@/components/logs/controls-container";
import { useNamespaceListFilters } from "../hooks/use-namespace-list-filters";
import { NamespaceListDateTime } from "./components/namespace-list-datetime";
import { NamespaceListRefresh } from "./components/namespace-list-refresh";

export function NamespaceListControls() {
  return (
    <ControlsContainer>
      <ControlsLeft>
        <ListSearchInput
          useFiltersHook={useNamespaceListFilters}
          placeholder="Search namespaces..."
        />
        <NamespaceListDateTime />
      </ControlsLeft>
      <ControlsRight>
        <NamespaceListRefresh />
      </ControlsRight>
    </ControlsContainer>
  );
}
