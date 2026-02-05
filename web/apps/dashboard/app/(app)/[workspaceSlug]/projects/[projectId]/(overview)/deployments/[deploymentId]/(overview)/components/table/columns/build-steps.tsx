import type { Column } from "@/components/virtual-table/types";
import { cn } from "@/lib/utils";
import { formatLatency } from "@/lib/utils/metric-formatters";
import type { BuildStep, BuildStepLog } from "@unkey/clickhouse/src/build-steps";
import { CaretUp } from "@unkey/icons";
import { Badge, TimestampInfo } from "@unkey/ui";

export type BuildStepRow = BuildStep & {
  logs?: Omit<BuildStepLog, "step_id">[];
  _isExpanded?: boolean;
};

export const buildStepsColumns: Column<BuildStepRow>[] = [
  {
    key: "started_at",
    header: "Started At",
    width: "180px",
    headerClassName: "pl-8",
    cellClassName: "pl-8",
    render: (step) => (
      <div
        className={cn(
          "font-mono text-xs truncate max-w-[300px] flex items-center gap-2",
          step.has_logs && "-ml-5",
        )}
        title={step.name}
      >
        {step.has_logs && (
          <CaretUp
            iconSize="sm-regular"
            className={cn(
              "shrink-0 transition-transform text-gray-9",
              step._isExpanded && "rotate-180",
            )}
          />
        )}
        <TimestampInfo
          value={step.started_at}
          className="font-mono group-hover:underline decoration-dotted"
        />
      </div>
    ),
  },
  {
    key: "name",
    header: "Step",
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
    key: "status",
    header: "Status",
    width: "120px",
    render: (step) => {
      if (step.error) {
        return (
          <Badge className="uppercase px-[6px] rounded-md font-mono whitespace-nowrap bg-error-4 text-error-11 group-hover:bg-error-5">
            Failed
          </Badge>
        );
      }
      if (step.cached) {
        return (
          <Badge className="uppercase px-[6px] rounded-md font-mono whitespace-nowrap bg-blue-4 text-blue-11 group-hover:bg-blue-5">
            Cached
          </Badge>
        );
      }
      return (
        <Badge className="uppercase px-[6px] rounded-md font-mono whitespace-nowrap bg-grayA-3 text-grayA-11 group-hover:bg-grayA-5">
          Success
        </Badge>
      );
    },
  },
  {
    key: "duration",
    header: "Duration",
    width: "120px",
    render: (step) => {
      const duration = step.completed_at - step.started_at;
      return (
        <span className="px-[6px] font-mono whitespace-nowrap tabular-nums">
          {formatLatency(duration)}
        </span>
      );
    },
  },

  {
    key: "logs",
    header: "Logs",
    width: "100px",
    render: (step) => {
      if (!step.has_logs) {
        return <span className="opacity-30">—</span>;
      }
      const logCount = step.logs?.length ?? 0;
      return (
        <div className="flex">
          {logCount > 0 ? (
            <span>
              <span className="text-accent-11 tabular-nums">{logCount}</span>
              <span> lines</span>
            </span>
          ) : (
            "—"
          )}
        </div>
      );
    },
  },
  {
    key: "error",
    header: "Error",
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
];
