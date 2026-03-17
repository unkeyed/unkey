import { RegionFlag } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/components/region-flag";
import type { Column } from "@/components/virtual-table/types";
import { mapRegionToFlag } from "@/lib/trpc/routers/deploy/network/utils";
import { TriangleWarning } from "@unkey/icons";
import { TimestampInfo } from "@unkey/ui";
import { TruncatedCell } from "../truncated-cell";

export type ContainerLogRow = {
  time: number;
  severity: string;
  message: string;
  instance_id: string;
  region: string;
};

function SeverityIcon({ severity }: { severity: string }) {
  switch (severity.toUpperCase()) {
    case "ERROR":
      return <TriangleWarning className="text-error-11" iconSize="md-regular" />;
    case "WARN":
      return <TriangleWarning className="text-warning-11" iconSize="md-regular" />;
    default:
      return null;
  }
}

export const containerLogColumns: Column<ContainerLogRow>[] = [
  {
    key: "log",
    width: "auto",
    cellClassName: "pl-[25px]",
    render: (log) => (
      <div className="flex items-center gap-6">
        <TimestampInfo
          displayType="local_hours_with_millis"
          value={log.time}
          className="font-mono group-hover:underline decoration-dotted"
        />
        <SeverityIcon severity={log.severity} />
        <div className="items-center flex gap-2">
          <RegionFlag flagCode={mapRegionToFlag(log.region)} size="xs" shape="circle" />
          <span className="font-mono text-xs uppercase">{log.region}</span>
        </div>
        <span className="font-mono truncate text-xs" title={log.instance_id}>
          {log.instance_id}
        </span>
        <div className="flex-1 min-w-0">
          <TruncatedCell
            text={log.message}
            threshold={120}
            maxWidth="max-w-[750px]"
            className="text-gray-12"
            side="top"
          />
        </div>
      </div>
    ),
  },
];
