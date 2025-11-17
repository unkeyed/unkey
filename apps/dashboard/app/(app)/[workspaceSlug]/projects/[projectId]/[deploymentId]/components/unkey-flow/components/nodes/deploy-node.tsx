import {
  Bolt,
  ChartActivity,
  CircleCheck,
  Focus,
  Heart,
  Layers3,
} from "@unkey/icons";
import { cn } from "@unkey/ui/src/lib/utils";
import {
  type DeploymentNode,
  type RegionMetadata,
  type HealthStatus,
  REGION_INFO,
} from "./types";
import { StatusDot } from "./status-dot";
import { HealthBanner } from "./health-banner";
import { STATUS_CONFIG } from "./status-config";
import type { PropsWithChildren } from "react";
import { InfoTooltip } from "@unkey/ui";

function getHealthStyles(health: HealthStatus): { ring: string; glow: string } {
  const styleMap: Record<HealthStatus, { ring: string; glow: string }> = {
    normal: {
      ring: "hover:ring-grayA-2",
      glow: "hover:shadow-[0_4px_16px_-4px_rgba(0,0,0,0.15),0_0_0_1px_hsl(var(--grayA-6)),0_0_30px_color-mix(in_srgb,hsl(var(--grayA-9))_17.5%,transparent)]",
    },
    unstable: {
      ring: "hover:ring-orangeA-3",
      glow: "hover:shadow-[0_4px_16px_-4px_rgba(0,0,0,0.15),0_0_0_1px_hsl(var(--orangeA-3)),0_0_30px_color-mix(in_srgb,hsl(var(--orangeA-9))_20%,transparent)]",
    },
    degraded: {
      ring: "hover:ring-warningA-3",
      glow: "hover:shadow-[0_4px_16px_-4px_rgba(0,0,0,0.15),0_0_0_1px_hsl(var(--warningA-3)),0_0_30px_color-mix(in_srgb,hsl(var(--warningA-9))_20%,transparent)]",
    },
    unhealthy: {
      ring: "hover:ring-errorA-3",
      glow: "hover:shadow-[0_4px_16px_-4px_rgba(0,0,0,0.15),0_0_0_1px_hsl(var(--errorA-3)),0_0_30px_color-mix(in_srgb,hsl(var(--errorA-9))_20%,transparent)]",
    },
    recovering: {
      ring: "hover:ring-featureA-3",
      glow: "hover:shadow-[0_4px_16px_-4px_rgba(0,0,0,0.15),0_0_0_1px_hsl(var(--featureA-3)),0_0_30px_color-mix(in_srgb,hsl(var(--featureA-9))_20%,transparent)]",
    },
    health_syncing: {
      ring: "hover:ring-infoA-3",
      glow: "hover:shadow-[0_4px_16px_-4px_rgba(0,0,0,0.15),0_0_0_1px_hsl(var(--infoA-3)),0_0_30px_color-mix(in_srgb,hsl(var(--infoA-9))_20%,transparent)]",
    },
    unknown: {
      ring: "hover:ring-grayA-2",
      glow: "hover:shadow-[0_4px_16px_-4px_rgba(0,0,0,0.15),0_0_0_1px_hsl(var(--grayA-6)),0_0_30px_color-mix(in_srgb,hsl(var(--grayA-9))_17.5%,transparent)]",
    },
    disabled: {
      ring: "hover:ring-grayA-2",
      glow: "hover:shadow-[0_4px_16px_-4px_rgba(0,0,0,0.15),0_0_0_1px_hsl(var(--grayA-6)),0_0_30px_color-mix(in_srgb,hsl(var(--grayA-9))_17.5%,transparent)]",
    },
  };
  return styleMap[health];
}

type StatusIndicatorProps = {
  icon: React.ReactNode;
  healthStatus: HealthStatus;
  tooltip: string;
  showGlow?: boolean;
};

function StatusIndicator({
  icon,
  healthStatus,
  tooltip,
  showGlow = false,
}: StatusIndicatorProps) {
  const { colors } = STATUS_CONFIG[healthStatus];
  const glowBoxShadow = showGlow
    ? `0 0 8px 1px ${colors.dotRing} inset, 0 0 0 1px var(--color-grayA-gray-a5, rgba(0, 9, 50, 0.12)) inset`
    : "";

  return (
    <InfoTooltip
      content={tooltip}
      variant="primary"
      className="px-2.5 py-1 rounded-[10px] text-whiteA-12 bg-blackA-12 text-xs z-30"
      position={{ align: "center", side: "top", sideOffset: 5 }}
    >
      <div
        className="border bg-gray-1 border-grayA-3 h-11 rounded-lg w-8 transition-all hover:ring-1 hover:ring-gray-7 duration-200 ease-out hover:scale-105 cursor-pointer"
        style={{
          boxShadow: glowBoxShadow,
        }}
      >
        <div className="h-6 border-b border-grayA-3 relative">
          <StatusDot healthStatus={healthStatus} />
        </div>
        <div className="h-5 bg-grayA-2 pl-1 pt-[3px]">{icon}</div>
      </div>
    </InfoTooltip>
  );
}

type MetricPillProps = {
  icon: React.ReactNode;
  value: string | number;
  tooltip: string;
};

function MetricPill({ icon, value, tooltip }: MetricPillProps) {
  return (
    <InfoTooltip
      content={tooltip}
      variant="primary"
      className="px-2.5 py-1 rounded-[10px] text-whiteA-12 bg-blackA-12 text-xs z-30"
      position={{ align: "center", side: "top", sideOffset: 5 }}
    >
      <div className="bg-grayA-3 p-1.5 flex items-center justify-between rounded-full h-5 gap-1.5 transition-all hover:bg-grayA-4 cursor-pointer">
        {icon}
        <span className="text-gray-9 text-[10px] tabular-nums">{value}</span>
      </div>
    </InfoTooltip>
  );
}

type CardHeaderProps = {
  icon: React.ReactNode;
  title: string;
  subtitle: string;
  health: HealthStatus;
};

function CardHeader({ icon, title, subtitle, health }: CardHeaderProps) {
  const { colors } = STATUS_CONFIG[health];

  return (
    <div
      className="border-b border-grayA-4 flex px-3 py-2.5 rounded-t-[14px]"
      style={{
        background:
          "radial-gradient(circle at 5% 15%, hsl(var(--grayA-3)) 0%, transparent 20%), light-dark(#FFF, #000)",
      }}
    >
      <div className="flex items-center justify-between gap-3">
        {icon}
        <div className="flex flex-col gap-[3px] justify-center h-9 py-2">
          <div className="text-accent-12 font-medium text-xs font-mono">
            {title}
          </div>
          <div className="text-gray-9 text-[11px]">{subtitle}</div>
        </div>
      </div>
      <div className="flex gap-2 items-center ml-auto">
        <StatusIndicator
          icon={<CircleCheck className="text-gray-9" iconSize="sm-regular" />}
          healthStatus="normal"
          tooltip="Gateway is online and serving traffic"
        />
        <StatusIndicator
          icon={<Heart className={colors.dotTextColor} iconSize="sm-regular" />}
          healthStatus={health}
          tooltip="Gateway health status"
          showGlow={health !== "normal"}
        />
      </div>
    </div>
  );
}

type RegionCardFooterProps = {
  type: "region";
  rps?: number;
  cpu?: number;
  memory?: number;
};

type InstanceCardFooterProps = {
  type: "instance";
  flagCode: RegionMetadata["flagCode"];
  rps?: number;
  cpu?: number;
  memory?: number;
};

type CardFooterProps = RegionCardFooterProps | InstanceCardFooterProps;

function CardFooter(props: CardFooterProps) {
  const { type, rps, cpu, memory } = props;
  const flagCode = type === "instance" ? props.flagCode : undefined;
  const isRegion = type === "region";

  return (
    <div className="p-1 flex items-center h-full bg-grayA-2 rounded-b-[14px]">
      {flagCode && (
        <div className="size-[22px] bg-grayA-3 rounded-full p-[3px] flex items-center justify-center mr-1.5">
          <img
            src={`/images/flags/${flagCode}.svg`}
            alt={flagCode}
            className="size-4"
          />
        </div>
      )}
      {rps !== undefined && (
        <MetricPill
          icon={<ChartActivity iconSize="sm-medium" className="shrink-0" />}
          value={rps}
          tooltip={
            isRegion
              ? "Requests per second handled by this region's gateways"
              : "Requests per second handled by this instance"
          }
        />
      )}
      <div className="flex items-center gap-2 ml-auto">
        {cpu !== undefined && (
          <MetricPill
            icon={<Bolt iconSize="sm-medium" className="shrink-0" />}
            value={`${cpu}%`}
            tooltip={
              isRegion
                ? "Average CPU usage across all gateway instances in this region"
                : "Current CPU usage for this instance"
            }
          />
        )}
        {memory !== undefined && (
          <MetricPill
            icon={<Focus iconSize="sm-regular" className="shrink-0" />}
            value={`${memory}%`}
            tooltip={
              isRegion
                ? "Average memory usage across all gateway instances in this region"
                : "Current memory usage for this instance"
            }
          />
        )}
      </div>
    </div>
  );
}

type NodeWrapperProps = PropsWithChildren<{
  health: HealthStatus;
}>;

function NodeWrapper({ health, children }: NodeWrapperProps) {
  const isDisabled = health === "disabled";
  const { ring, glow } = getHealthStyles(health);

  return (
    <div
      className={cn(
        "relative w-[282px] rounded-[14px]",
        isDisabled
          ? "grayscale opacity-90 cursor-not-allowed"
          : cn(
              "hover:scale-[1.001] transition-all duration-200 ease-out cursor-pointer",
              "hover:ring-2 hover:ring-offset-0",
              ring,
              glow
            )
      )}
    >
      <HealthBanner healthStatus={health} />
      <div
        className={cn(
          "relative z-20 w-[282px] h-[100px] border border-grayA-4 rounded-[14px] flex flex-col bg-white dark:bg-black shadow-[0_2px_8px_-2px_rgba(0,0,0,0.1)]"
        )}
      >
        {children}
      </div>
    </div>
  );
}

type RegionNodeProps = {
  node: DeploymentNode & { metadata: { type: "region" } };
};

export function RegionNode({ node }: RegionNodeProps) {
  const { flagCode, rps, cpu, memory, health, zones } = node.metadata;
  const regionInfo = REGION_INFO[flagCode];

  return (
    <NodeWrapper health={health}>
      <CardHeader
        icon={
          <InfoTooltip
            content={`AWS region ${node.label} (${regionInfo.location})`}
            variant="primary"
            className="px-2.5 py-1 rounded-[10px] bg-white dark:bg-blackA-12 text-xs z-30"
            position={{ align: "center", side: "top", sideOffset: 5 }}
          >
            <div className="border rounded-[10px] border-grayA-3 size-9 bg-grayA-3 flex items-center justify-center">
              <img
                src={`/images/flags/${flagCode}.svg`}
                alt={flagCode}
                className="size-4"
              />
            </div>
          </InfoTooltip>
        }
        title={node.label}
        subtitle={`${zones} availability ${zones === 1 ? "zone" : "zones"}`}
        health={health}
      />
      <CardFooter type="region" rps={rps} cpu={cpu} memory={memory} />
    </NodeWrapper>
  );
}

type InstanceNodeProps = {
  node: DeploymentNode & { metadata: { type: "instance" } };
  flagCode: RegionMetadata["flagCode"];
};

export function InstanceNode({ node, flagCode }: InstanceNodeProps) {
  const { rps, cpu, memory, health } = node.metadata;

  return (
    <NodeWrapper health={health}>
      <CardHeader
        icon={
          <div className="border rounded-[10px] size-9 flex items-center justify-center border-grayA-5 bg-grayA-2">
            <Layers3 iconSize="lg-medium" className="text-gray-11" />
          </div>
        }
        title={node.label}
        subtitle="Instance Replica"
        health={health}
      />
      <CardFooter
        type="instance"
        flagCode={flagCode}
        rps={rps}
        cpu={cpu}
        memory={memory}
      />
    </NodeWrapper>
  );
}
