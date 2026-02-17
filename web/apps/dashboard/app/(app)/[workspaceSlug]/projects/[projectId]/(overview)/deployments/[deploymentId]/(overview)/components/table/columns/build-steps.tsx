import type { Column } from "@/components/virtual-table/types";
import { cn } from "@/lib/utils";
import { formatLatency } from "@/lib/utils/metric-formatters";
import type { BuildStep, BuildStepLog } from "@unkey/clickhouse/src/build-steps";
import { Bolt, CaretRight, TriangleWarning } from "@unkey/icons";
import { InfoTooltip, TimestampInfo } from "@unkey/ui";

export type BuildStepRow = BuildStep & {
  logs?: Omit<BuildStepLog, "step_id">[];
  _isExpanded?: boolean;
};

export const buildStepsColumns: Column<BuildStepRow>[] = [
  {
    key: "expand",
    width: "32px",
    render: (step) =>
      step.has_logs ? (
        <div className="size-4 flex items-center justify-center ">
          <CaretRight
            iconSize="sm-regular"
            className={cn(
              "shrink-0 transition-transform text-gray-9  ",
              step._isExpanded && "rotate-90",
            )}
          />
        </div>
      ) : null,
  },
  {
    key: "started_at",
    //header: "Started At",
    width: "180px",
    render: (step) => (
      <div className="font-mono text-xs truncate max-w-[300px] flex items-center gap-2 ">
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
    render: (step) => {
      if (step.error) {
        return <TriangleWarning className="text-error-11" />;
      }
      if (step.cached) {
        return (
          <InfoTooltip content="This step was cached" asChild>
            <Bolt className="text-primary-11" />
          </InfoTooltip>
        );
      }
      return null;
    },
  },
  {
    key: "name",
    // header: "Step",
    width: "250px",
    render: (step) => (
      <div
        className="font-mono text-xs truncate max-w-[300px] flex items-center gap-2"
        title={step.name}
      >
        <span className="truncate">{step.name}</span>
      </div>
    ),
  },

  {
    key: "error",
    //header: "Error",
    width: "300px",
    render: (step) => {
      if (!step.error) {
        return null;
      }
      return (
        <span className="truncate font-mono" title={step.error}>
          {step.error}
        </span>
      );
    },
  },
  {
    key: "duration",
    //header: "Duration",
    width: "10%",
    render: (step) => {
      const duration = step.completed_at - step.started_at;
      return (
        <span className="px-[6px] font-mono whitespace-nowrap tabular-nums">
          {formatLatency(duration)}
        </span>
      );
    },
  },
];
