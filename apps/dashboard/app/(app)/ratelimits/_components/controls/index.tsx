import {
  ControlsContainer,
  ControlsLeft,
  ControlsRight,
} from "@/components/logs/controls-container";
import { LogsDateTime } from "./components/logs-datetime";
import { LogsRefresh } from "./components/logs-refresh";
import { LogsSearch } from "./components/logs-search";

type RatelimitListControlsProps = {
  setNamespaces: (namespaces: { id: string; name: string }[]) => void;
  initialNamespaces: { id: string; name: string }[];
};

export function RatelimitListControls({
  setNamespaces,
  initialNamespaces,
}: RatelimitListControlsProps) {
  return (
    <ControlsContainer>
      <ControlsLeft>
        <LogsSearch setNamespaces={setNamespaces} initialNamespaces={initialNamespaces} />
        <LogsDateTime />
      </ControlsLeft>
      <ControlsRight>
        <LogsRefresh />
      </ControlsRight>
    </ControlsContainer>
  );
}
