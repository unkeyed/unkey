import {
  ControlsContainer,
  ControlsLeft,
  ControlsRight,
} from "@/components/logs/controls-container";
import { NamespaceListDateTime } from "./components/namespace-list-datetime";
import { NamespaceListRefresh } from "./components/namespace-list-refresh";
import { NamespaceSearchInput } from "./components/namespace-list-search";

export function NamespaceListControls() {
  return (
    <ControlsContainer>
      <ControlsLeft>
        <NamespaceSearchInput />
        <NamespaceListDateTime />
      </ControlsLeft>
      <ControlsRight>
        <NamespaceListRefresh />
      </ControlsRight>
    </ControlsContainer>
  );
}
