import { cn } from "@/lib/utils";
import { TimestampInfo } from "@unkey/ui";
import { Fragment } from "react/jsx-runtime";
import { TruncatedCell } from "../truncated-cell";
import type { BuildStepRow } from "./columns";

export function BuildStepLogsExpanded({ step }: { step: BuildStepRow }) {
  if (!step.logs || step.logs.length === 0) {
    return (
      <tr>
        <td colSpan={6} className="px-8 py-4 text-sm text-gray-11">
          No logs available for this step
        </td>
      </tr>
    );
  }

  const isError = Boolean(step.error);
  const borderClass = isError ? "border-error-7" : "border-accent-7";
  const bgClass = isError ? "bg-error-2" : "";

  return (
    <>
      <tr>
        <td colSpan={6} className={cn("border-l-2 p-0", borderClass, bgClass)} />
      </tr>
      {step.logs.map((log, idx) => (
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
