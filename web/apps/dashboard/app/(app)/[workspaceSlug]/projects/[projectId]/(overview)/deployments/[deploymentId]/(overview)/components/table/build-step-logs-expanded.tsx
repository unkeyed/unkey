import { CopyButton, TimestampInfo } from "@unkey/ui";
import { format } from "date-fns";
import type { BuildStepRow } from "./columns/build-steps";

export function BuildStepLogsExpanded({ step }: { step: BuildStepRow }) {
  if (!step.logs || step.logs.length === 0) {
    return <div className="px-8 py-4 text-sm text-gray-9">No logs available for this step</div>;
  }

  const logsContent = step.logs
    .map((log) => `[${format(new Date(log.time), "HH:mm:ss.SSS")}] ${log.message}`)
    .join("\n");

  return (
    <div className="px-8 py-3">
      <div className="border bg-gray-2 border-gray-4 rounded-[10px] relative group">
        {/* Title bar */}
        <div className="text-gray-11 text-xs leading-6 px-[14px] py-1.5 font-sans flex items-center justify-between">
          <span>Step Output</span>
        </div>

        {/* Log content */}
        <div className="border-gray-4 border-t rounded-b-[10px] bg-white dark:bg-black px-3.5 py-2 max-h-[400px] overflow-y-auto group-[timestamp]">
          <pre className="whitespace-pre-wrap break-words text-xs text-accent-12 font-mono leading-relaxed">
            {step.logs.map((log, idx) => (
              <div
                key={`${log.time}-${idx}`}
                className="flex gap-3 hover:bg-gray-2 dark:hover:bg-gray-3 px-1 py-0.5 rounded-sm transition-colors"
              >
                <TimestampInfo value={log.time} className="font-mono" />
                <span className="text-accent-12 break-all">{log.message}</span>
              </div>
            ))}
          </pre>
        </div>

        {/* Copy button (appears on hover) */}
        <CopyButton
          value={logsContent}
          shape="square"
          variant="outline"
          className="absolute bottom-2 right-2 opacity-0 group-hover:opacity-100 transition-opacity rounded-md p-4 bg-gray-2 hover:bg-gray-2 size-2"
          aria-label="Copy logs"
        />
      </div>
    </div>
  );
}
