import { ControlsContainer, ControlsLeft } from "@/components/logs/controls-container";
import { RootKeysFilters } from "./components/root-keys-filters";
import { RootKeysSearch } from "./components/root-keys-search";

export function RootKeysListControls() {
  return (
    <ControlsContainer>
      <ControlsLeft>
        <RootKeysSearch />
        <RootKeysFilters />
      </ControlsLeft>
    </ControlsContainer>
  );
}
