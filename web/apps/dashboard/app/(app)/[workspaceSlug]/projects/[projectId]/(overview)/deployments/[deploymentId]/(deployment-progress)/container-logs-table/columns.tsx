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
      return (
        <div className="my-2">
          <TriangleWarning className="text-error-11" iconSize="md-regular" />
        </div>
      );
    case "WARN":
      return (
        <div className="my-2">
          <TriangleWarning className="text-warning-11" iconSize="md-regular" />
        </div>
      );
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
      <div className="font-mono text-xs my-2">
        <TimestampInfo
          displayType="local_hours_with_millis"
          value={log.time}
          className="font-mono group-hover:underline decoration-dotted"
        />
      </div>
    ),
  },
  {
    key: "severity",
    width: "32px",
    render: (log) => <SeverityIcon severity={log.severity} />,
  },
  {
    key: "message",
    width: "auto",
    render: (log) => (
      <TruncatedCell
        text={log.message}
        threshold={120}
        maxWidth="max-w-[750px]"
        className="text-gray-12"
        side="top"
      />
    ),
  },
];
