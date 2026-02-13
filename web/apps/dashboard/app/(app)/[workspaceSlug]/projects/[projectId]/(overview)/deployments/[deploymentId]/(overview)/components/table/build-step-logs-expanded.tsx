import { TimestampInfo } from "@unkey/ui";
import type { BuildStepRow } from "./columns/build-steps";

export function BuildStepLogsExpanded({ step }: { step: BuildStepRow }) {
  if (!step.logs || step.logs.length === 0) {
    return <div className="px-8 py-4 text-sm text-gray-9">No logs available for this step</div>;
  }

  return (
    <div className="py-1 ml-[34px]">
      <pre className="whitespace-pre-wrap wrap-break-word text-xs font-mono leading-relaxed">
        {step.logs.map((log, idx) => (
          <div key={`${log.time}-${idx}`} className="flex gap-13 py-0.5">
            <TimestampInfo
              displayType="local_hours_with_millis"
              value={log.time}
              className="font-mono text-xs text-grayA-9 hover:underline decoration-dotted"
            />
            <span className="text-accent-12 break-all">{log.message}</span>
          </div>
        ))}
      </pre>
    </div>
  );
}
