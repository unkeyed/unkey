import { ListSearchInput } from "@/components/list-search-input";
import { ControlsContainer, ControlsLeft } from "@/components/logs/controls-container";
import { useProjectsFilters } from "../hooks/use-projects-filters";

export function ProjectsListControls() {
  return (
    <ControlsContainer>
      <ControlsLeft>
        <ListSearchInput useFiltersHook={useProjectsFilters} placeholder="Search projects..." />
      </ControlsLeft>
    </ControlsContainer>
  );
}
