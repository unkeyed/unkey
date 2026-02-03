"use client";

import { LogDetails as SharedLogDetails } from "@/components/logs/details/log-details";
import { LogSection } from "@/components/logs/details/log-details/components/log-section";
import { useRuntimeLogs } from "../../../context/runtime-logs-provider";
import { formatTimestamp, safeParseAttributes } from "../../../utils";
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

  const attributes = safeParseAttributes(log);

  return (
    <SharedLogDetails distanceToTop={distanceToTop} log={log} onClose={handleClose} >
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
                <span className="font-mono">{formatTimestamp(log.time)}</span>
              </div>
              <div>
                <span className="text-grayA-11">Severity:</span>{" "}
                <span className="font-mono">{log.severity}</span>
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

        {attributes && (
          <LogSection
            title="Attributes"
            details={<pre className="text-xs">{JSON.stringify(attributes, null, 2)}</pre>}
          />
        )}
      </SharedLogDetails.CustomSections>
    </SharedLogDetails>
  );
}
