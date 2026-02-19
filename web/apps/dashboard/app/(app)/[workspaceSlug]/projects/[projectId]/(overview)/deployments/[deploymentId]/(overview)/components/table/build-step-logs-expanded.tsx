import { TimestampInfo } from "@unkey/ui";
import type { BuildStepRow } from "./columns/build-steps";

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
      {step.logs.map((log, idx) => (
        <tr key={`${log.time}-${idx}`}>
          {/* expand col spacer */}
          <td className="p-0" />
          {/* timestamp — aligned with started_at */}
          <td className="p-0 py-0.5 align-top">
            <TimestampInfo
              displayType="local_hours_with_millis"
              value={log.time}
              className="font-mono text-xs text-grayA-9 hover:underline decoration-dotted"
            />
          </td>
          {/* status col spacer */}
          <td className="p-0" />
          {/* message — spans remaining columns */}
          <td colSpan={3} className="p-0 py-0.5 align-top">
            <span className="text-accent-12 break-all whitespace-pre-wrap font-mono text-xs leading-relaxed">
              {log.message}
            </span>
          </td>
        </tr>
      ))}
    </>
  );
}
