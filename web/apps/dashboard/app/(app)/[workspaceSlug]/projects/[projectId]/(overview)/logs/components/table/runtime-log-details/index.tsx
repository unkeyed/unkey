"use client";

import { LogDetails as SharedLogDetails } from "@/components/logs/details/log-details";
import { LogSection } from "@/components/logs/details/log-details/components/log-section";
import { TimestampInfo } from "@unkey/ui";
import { cn } from "@unkey/ui/src/lib/utils";
import { useRuntimeLogs } from "../../../context/runtime-logs-provider";
import { RuntimeLogHeader } from "./runtime-log-header";

type Props = {
  distanceToTop: number;
};

export function RuntimeLogDetails({ distanceToTop }: Props) {
  const { setSelectedLog, selectedLog: log } = useRuntimeLogs();

  if (!log) {
    return null;
  }

  const handleClose = () => {
    setSelectedLog(null);
  };

  return (
    <SharedLogDetails distanceToTop={distanceToTop} log={log} onClose={handleClose}>
      <SharedLogDetails.Section>
        <RuntimeLogHeader log={log} onClose={handleClose} />
      </SharedLogDetails.Section>

      <SharedLogDetails.CustomSections>
        <LogSection
          title="Log Information"
          details={
            <div className="space-y-2 text-xs">
              <div>
                <span className="text-grayA-11">Time:</span>{" "}
                <TimestampInfo
                  value={log.time}
                  className={cn("font-mono underline decoration-dotted")}
                />
              </div>
              <div>
                <span className="text-grayA-11">Severity:</span>{" "}
                <span className="font-mono uppercase">{log.severity}</span>
              </div>
              <div>
                <span className="text-grayA-11">Message:</span>{" "}
                <span className="font-mono">{log.message}</span>
              </div>
            </div>
          }
        />

        <LogSection
          title="Deployment Info"
          details={
            <div className="space-y-2 text-xs">
              <div>
                <span className="text-grayA-11">Deployment ID:</span>{" "}
                <span className="font-mono">{log.deployment_id}</span>
              </div>
              <div>
                <span className="text-grayA-11">Region:</span>{" "}
                <span className="font-mono">{log.region}</span>
              </div>
            </div>
          }
        />

        {log.attributes && (
          <LogSection
            title="Attributes"
            details={<pre className="text-xs">{JSON.stringify(log.attributes, null, 2)}</pre>}
          />
        )}
      </SharedLogDetails.CustomSections>
    </SharedLogDetails>
  );
}
