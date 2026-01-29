import { Bolt, ChartActivity, Focus } from "@unkey/icons";
import type { SentinelNode } from "../types";
import { MetricPill } from "./metric-pill";

type SentinelCardFooterProps = {
  type: "sentinel";
  rps?: number;
  cpu?: number;
  memory?: number;
};

type InstanceCardFooterProps = {
  type: "instance";
  flagCode: SentinelNode["metadata"]["flagCode"];
  rps?: number;
  cpu?: number;
  memory?: number;
};

type CardFooterProps = SentinelCardFooterProps | InstanceCardFooterProps;

export function CardFooter(props: CardFooterProps) {
  const { type, rps, cpu, memory } = props;
  const flagCode = type === "instance" ? props.flagCode : undefined;
  const isSentinel = type === "sentinel";

  return (
    <div className="p-1 flex items-center h-full bg-grayA-2 rounded-b-[14px]">
      {flagCode && (
        <div className="size-[22px] bg-grayA-3 rounded-full p-[3px] flex items-center justify-center mr-1.5">
          <img src={`/images/flags/${flagCode}.svg`} alt={flagCode} className="size-4" />
        </div>
      )}
      {rps !== undefined && (
        <MetricPill
          icon={<ChartActivity iconSize="sm-medium" className="shrink-0" />}
          value={rps}
          tooltip={
            isSentinel
              ? "Requests per second handled by this sentinel"
              : "Requests per second handled by this instance"
          }
        />
      )}
      <div className="flex items-center gap-2 ml-auto">
        {cpu !== undefined && (
          <MetricPill
            icon={<Bolt iconSize="sm-medium" className="shrink-0" />}
            value={formatCpu(cpu)}
            tooltip={
              isSentinel ? "CPU allocated to this sentinel" : "CPU allocated to this instance"
            }
          />
        )}
        {memory !== undefined && (
          <MetricPill
            icon={<Focus iconSize="sm-regular" className="shrink-0" />}
            value={formatMemory(memory)}
            tooltip={
              isSentinel ? "Memory allocated to this sentinel" : "Memory allocated to this instance"
            }
          />
        )}
      </div>
    </div>
  );
}

function formatCpu(millicores: number): string {
  if (millicores >= 1000) {
    return `${(millicores / 1000).toFixed(millicores % 1000 === 0 ? 0 : 1)}c`;
  }
  return `${millicores}m`;
}

function formatMemory(mib: number): string {
  if (mib >= 1024) {
    return `${(mib / 1024).toFixed(mib % 1024 === 0 ? 0 : 1)} GiB`;
  }
  return `${mib} MiB`;
}
