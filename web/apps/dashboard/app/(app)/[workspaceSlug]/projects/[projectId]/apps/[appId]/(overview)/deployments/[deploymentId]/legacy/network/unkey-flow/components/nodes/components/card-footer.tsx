import { RegionFlag } from "@/components/region-flag";
import { formatCpuParts, formatMemoryParts } from "@/lib/utils/deployment-formatters";
import { IconChartActivityOutline12, IconMicrochipOutline18, IconRamOutline18 } from "@unkey/icons";
import { MetricPill } from "./metric-pill";

// Region cards show only an aggregate RPS pill; per-resource metrics
// (cpu, memory) live on the child instance nodes.
type RegionCardFooterProps = {
  type: "region";
  rps?: number;
};

type InstanceCardFooterProps = {
  type: "instance";
  region: string;
  rps?: number;
  cpu?: number;
  memory?: number;
};

type CardFooterProps = RegionCardFooterProps | InstanceCardFooterProps;

export function CardFooter(props: CardFooterProps) {
  const rps = props.rps;
  const region = props.type === "instance" ? props.region : undefined;
  const cpu = props.type === "instance" ? props.cpu : undefined;
  const memory = props.type === "instance" ? props.memory : undefined;

  return (
    <div className="p-1 flex items-center h-full bg-grayA-2 rounded-b-2xl">
      {region && <RegionFlag region={region} size="sm" shape="circle" className="mr-1.5" />}
      {rps !== undefined && (
        <MetricPill
          icon={<IconChartActivityOutline12 className="shrink-0" />}
          value={formatRps(rps)}
          tooltip="Avg. RPS over last 15 min (updated every 5s)"
        />
      )}
      <div className="flex items-center gap-2 ml-auto">
        {cpu !== undefined &&
          (() => {
            const parts = formatCpuParts(cpu);
            return (
              <MetricPill
                icon={<IconMicrochipOutline18 className="size-3 shrink-0" />}
                value={
                  <>
                    <span className="font-medium">{parts.value}</span> {parts.unit}
                  </>
                }
                tooltip="CPU allocated to this instance"
              />
            );
          })()}
        {memory !== undefined &&
          (() => {
            const parts = formatMemoryParts(memory);
            return (
              <MetricPill
                icon={<IconRamOutline18 className="size-3 shrink-0" />}
                value={
                  <>
                    <span className="font-medium">{parts.value}</span> {parts.unit}
                  </>
                }
                tooltip="Memory allocated to this instance"
              />
            );
          })()}
      </div>
    </div>
  );
}

function formatRps(rps: number): string {
  return `${rps} RPS`;
}
