import type { KeyDetailsLog } from "@unkey/clickhouse/src/verifications";
import { InfoTooltip } from "@unkey/ui";
import { STATUS_STYLES } from "../utils/get-row-class";
import {
  LOG_OUTCOME_DEFINITIONS,
  type LogOutcomeType,
  getStatusType,
} from "../utils/outcome-definitions";
import { StatusBadge } from "./status-badge";

type OutcomeCellProps = {
  log: KeyDetailsLog;
  isSelected: boolean;
};

export const OutcomeCell = ({ log, isSelected }: OutcomeCellProps) => {
  const outcomeType: LogOutcomeType =
    log.outcome in LOG_OUTCOME_DEFINITIONS ? (log.outcome as LogOutcomeType) : "";
  const outcomeInfo = LOG_OUTCOME_DEFINITIONS[outcomeType];

  return (
    <InfoTooltip
      variant="inverted"
      className="cursor-default"
      content={<p>{outcomeInfo.tooltip}</p>}
      position={{ side: "top", align: "center", sideOffset: 5 }}
    >
      <div className="flex gap-3 items-center">
        <StatusBadge
          primary={{
            label: outcomeInfo.label,
            color: isSelected
              ? STATUS_STYLES[getStatusType(outcomeInfo.type)].badge.selected
              : STATUS_STYLES[getStatusType(outcomeInfo.type)].badge.default,
            icon: outcomeInfo.icon,
          }}
        />
      </div>
    </InfoTooltip>
  );
};
