import { LogsDateTime } from "@/app/(app)/[workspaceSlug]/apis/_components/controls/components/logs-datetime";
import { ControlsContainer, ControlsLeft } from "@/components/logs/controls-container";
import { LogsFilters } from "./components/logs-filters";
import { LogsSearch } from "./components/logs-search";

export function KeysListControls({ keyspaceId }: { keyspaceId: string }) {
  return (
    <ControlsContainer>
      <ControlsLeft>
        <LogsSearch keyspaceId={keyspaceId} />
        <LogsFilters />
        <LogsDateTime />
      </ControlsLeft>
    </ControlsContainer>
  );
}
