import {
  ControlsContainer,
  ControlsLeft,
  ControlsRight,
} from "@/components/logs/controls-container";
import type { ApiOverview } from "@/lib/trpc/routers/api/overview/query-overview/schemas";
import { LogsDateTime } from "./components/logs-datetime";
import { LogsRefresh } from "./components/logs-refresh";
import { LogsSearch } from "./components/logs-search";

type Props = {
  apiList: ApiOverview[];
  onApiListChange: (apiList: ApiOverview[]) => void;
  onSearch: (value: boolean) => void;
};

export function ApiListControls(props: Props) {
  return (
    <ControlsContainer>
      <ControlsLeft>
        <LogsSearch {...props} />
        <LogsDateTime />
      </ControlsLeft>
      <ControlsRight>
        <LogsRefresh />
      </ControlsRight>
    </ControlsContainer>
  );
}
