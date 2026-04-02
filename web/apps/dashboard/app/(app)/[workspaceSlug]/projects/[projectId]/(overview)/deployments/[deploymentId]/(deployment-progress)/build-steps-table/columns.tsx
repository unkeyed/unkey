import type { Column } from "@/components/virtual-table/types";
import { cn } from "@/lib/utils";
import { formatLatency } from "@/lib/utils/metric-formatters";
import type { BuildStep, BuildStepLog } from "@unkey/clickhouse/src/build-steps";
import { Bolt, CaretRight, TriangleWarning } from "@unkey/icons";
import { InfoTooltip, TimestampInfo } from "@unkey/ui";
import { TruncatedCell } from "../truncated-cell";

export type BuildStepRow = BuildStep & {
  logs?: Omit<BuildStepLog, "step_id">[];
  _isExpanded?: boolean;
};

export const buildStepsColumns: Column<BuildStepRow>[] = [
  {
    key: "expand",
    width: "25px",
    cellClassName: "p-0 align-top",
    render: (step) =>
      step.has_logs ? (
        <div className="my-2 size-4 flex items-center justify-center w-full shrink-0">
          <CaretRight
            iconSize="sm-regular"
            className={cn(
              "shrink-0 transition-transform text-gray-11",
              step._isExpanded && "rotate-90",
            )}
          />
        </div>
      ) : null,
  },
  {
    key: "started_at",
    width: "85px",
    cellClassName: "align-top",
    render: (step) => (
      <div className="font-mono text-xs truncate max-w-75 my-2">
        <TimestampInfo
          displayType="local_hours_with_millis"
          value={step.started_at}
          className="font-mono group-hover:underline decoration-dotted"
        />
      </div>
    ),
  },
  {
    key: "status",
    width: "32px",
    cellClassName: "align-top",
    render: (step) => {
      if (step.error) {
        return (
          <div className="my-2">
            <TriangleWarning className="text-error-11" iconSize="md-regular" />
          </div>
        );
      }
      if (step.cached) {
        return (
          <div className="my-2">
            <InfoTooltip content="This step was cached" asChild>
              <Bolt className="text-primary-11" iconSize="md-regular" />
            </InfoTooltip>
          </div>
        );
      }
      return null;
    },
  },
  {
    key: "name",
    width: "400px",
    cellClassName: "align-top",
    render: (step) => (
      <TruncatedCell
        text={step.name}
        className="flex items-center gap-2 text-gray-12"
      />
    ),
  },
  {
    key: "error",
    width: "400px",
    cellClassName: "align-top",
    render: (step) => {
      if (!step.error) {
        return null;
      }
      return <TruncatedCell text={step.error} />;
    },
  },
  {
    key: "duration",
    width: "115px",
    cellClassName: "align-top",
    render: (step) => {
      const duration = step.completed_at - step.started_at;
      return (
        <div className="my-2 flex justify-end font-mono whitespace-nowrap tabular-nums">
          {formatLatency(duration)}
        </div>
      );
    },
  },
];
