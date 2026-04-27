"use client";

import { DeploymentIdLink } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/components/deployment-id-link";
import { RegionFlag } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/components/region-flag";
import { LogDetails as SharedLogDetails } from "@/components/logs/details/log-details";
import { LogSection } from "@/components/logs/details/log-details/components/log-section";
import { mapRegionToFlag } from "@/lib/trpc/routers/deploy/network/utils";
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
    <SharedLogDetails distanceToTop={distanceToTop} log={log} onClose={handleClose} animated>
      <SharedLogDetails.Section>
        <RuntimeLogHeader log={log} onClose={handleClose} />
      </SharedLogDetails.Section>

      <SharedLogDetails.CustomSections>
        <LogSection
          title="Log Information"
          details={
            <div className="flex flex-col gap-2 text-xs">
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
            <div className="flex flex-col gap-2 text-xs">
              <div className="flex items-center gap-1.5">
                <span className="text-grayA-11">Region:</span>{" "}
                <RegionFlag flagCode={mapRegionToFlag(log.region)} size="xs" shape="circle" />
                <span className="font-mono">{log.region}</span>
              </div>
              <div>
                <span className="text-grayA-11">Instance ID:</span>{" "}
                <span className="font-mono">{log.instance_id}</span>
              </div>
              <div className="flex items-center gap-1.5">
                <span className="text-grayA-11">Deployment ID:</span>{" "}
                <DeploymentIdLink deploymentId={log.deployment_id} />
              </div>
            </div>
          }
        />

        {log.attributes && (
          <LogSection title="Attributes" details={JSON.stringify(log.attributes, null, 2)} />
        )}
      </SharedLogDetails.CustomSections>
    </SharedLogDetails>
  );
}
