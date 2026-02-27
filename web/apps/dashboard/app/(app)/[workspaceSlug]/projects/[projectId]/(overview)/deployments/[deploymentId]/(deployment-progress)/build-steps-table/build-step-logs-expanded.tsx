import { TimestampInfo } from "@unkey/ui";
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
      <tr aria-hidden="true">
        <td colSpan={6} className="p-0" />
      </tr>
      {step.logs.map((log, idx) => (
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
          <td colSpan={3} className="py-0">
            <span className="whitespace-pre-wrap font-mono text-xs leading-snug text-gray-12 break-all ml-5">
              {log.message}
            </span>
          </td>
        </tr>
      ))}
    </>
  );
}
