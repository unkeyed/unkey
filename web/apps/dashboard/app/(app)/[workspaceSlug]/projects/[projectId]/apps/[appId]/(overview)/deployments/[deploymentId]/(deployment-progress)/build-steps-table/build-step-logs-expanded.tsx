import { TimestampInfo } from "@unkey/ui";
import { cn } from "cn";
import { Fragment } from "react/jsx-runtime";
import { useDeployment } from "../../layout-provider";
import { BUILD_STEP_LOG_ENTRIES_SHOWN_MAX, useBuildStepLogs } from "../../use-build-step-logs";
import { TruncatedCell } from "../truncated-cell";
import type { BuildStepRow } from "./columns";

export function BuildStepLogsExpanded({ step }: { step: BuildStepRow }) {
  const { deployment } = useDeployment();
  const logs = useBuildStepLogs(deployment, step.step_id);

  if (logs.isLoading) {
    return <BuildStepLogsMessage text="Loading logs" />;
  }
  if (logs.isError) {
    return <BuildStepLogsMessage text="Failed to load logs for this step" />;
  }
  if (logs.data.entries.length === 0) {
    return <BuildStepLogsMessage text="No logs available for this step" />;
  }

  const isError = Boolean(step.error);
  const borderClass = isError ? "border-error-7" : "border-strong";
  const bgClass = isError ? "bg-error-2" : "";

  return (
    <>
      <tr>
        <td colSpan={6} className={cn("border-l-2 p-0", borderClass, bgClass)} />
      </tr>
      {logs.data.entriesTotal > BUILD_STEP_LOG_ENTRIES_SHOWN_MAX && (
        <tr>
          <td className={cn("border-l-2 py-0", borderClass, bgClass)} />
          <td colSpan={5} className={cn("py-1 text-xs text-gray-11", bgClass)}>
            Showing the last {BUILD_STEP_LOG_ENTRIES_SHOWN_MAX} of{" "}
            {logs.data.entriesTotal.toLocaleString()} entries
          </td>
        </tr>
      )}
      {logs.data.entries.map((log, idx) => (
        <Fragment key={`row-group-${log.time}-${idx}`}>
          <tr key={`spacer-${log.time}-${idx}`} style={{ height: "4px" }}>
            <td colSpan={6} className={cn("border-l-2 p-0", borderClass, bgClass)} />
          </tr>
          <tr key={`${log.time}-${idx}`}>
            <td className={cn("border-l-2 py-0", borderClass, bgClass)} />
            <td className={cn("py-0", bgClass)}>
              <TimestampInfo
                displayType="local_hours_with_millis"
                value={log.time}
                className="font-mono text-xs text-grayA-9 hover:underline decoration-dotted"
              />
            </td>
            <td className={cn("py-0", bgClass)} />
            <td colSpan={3} className={cn("h-[26px] py-px", bgClass)}>
              <TruncatedCell text={log.message} />
            </td>
          </tr>
        </Fragment>
      ))}
    </>
  );
}

function BuildStepLogsMessage({ text }: { text: string }) {
  return (
    <tr>
      <td colSpan={6} className="px-8 py-4 text-sm text-gray-11">
        {text}
      </td>
    </tr>
  );
}
