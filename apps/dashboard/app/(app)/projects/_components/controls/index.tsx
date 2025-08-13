import { ControlsContainer, ControlsLeft } from "@/components/logs/controls-container";
import { ProjectsSearchInput } from "./components/projects-list-search";

export function ProjectsListControls() {
  return (
    <ControlsContainer>
      <ControlsLeft>
        <ProjectsSearchInput />
      </ControlsLeft>
    </ControlsContainer>
  );
}
