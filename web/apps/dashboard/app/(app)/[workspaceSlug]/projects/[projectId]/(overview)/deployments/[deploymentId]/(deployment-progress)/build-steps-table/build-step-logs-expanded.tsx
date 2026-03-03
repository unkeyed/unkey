import { TimestampInfo } from "@unkey/ui";
import { Fragment } from "react/jsx-runtime";
import type { BuildStepRow } from "./columns";

export function BuildStepLogsExpanded({ step }: { step: BuildStepRow }) {
  if (!step.logs || step.logs.length === 0) {
    return (
      <tr>
        <td colSpan={6} className="px-8 py-4 text-sm text-gray-9">
          No logs available for this step
        </td>
      </tr>
    );
  }

  return (
    <>
      <tr>
        <td className="border-l-2 border-accent-7 p-0" />
      </tr>
      {step.logs.map((log, idx) => (
        <Fragment key={`row-group-${log.time}-${idx}`}>
          <tr key={`spacer-${log.time}-${idx}`} style={{ height: "4px" }}>
            <td className="border-l-2 border-accent-7 p-0" />
          </tr>
          <tr key={`${log.time}-${idx}`}>
            <td className="border-l-2 border-accent-7 py-0" />
            <td className="py-0">
              <TimestampInfo
                displayType="local_hours_with_millis"
                value={log.time}
                className="font-mono text-xs text-grayA-9 hover:underline decoration-dotted"
              />
            </td>
            <td className="py-0" />
            <td colSpan={3} className="h-[26px] py-[1px]">
              <span className="whitespace-pre-wrap font-mono text-xs text-gray-12 break-all">
                {log.message}
              </span>
            </td>
          </tr>
        </Fragment>
      ))}
    </>
  );
}
