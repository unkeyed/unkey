import {
  Bolt,
  ChartActivity,
  CircleCheck,
  Focus,
  Heart,
  Layers3,
  TriangleWarning,
  TriangleWarning2,
} from "@unkey/icons";
import { cn } from "@unkey/ui/src/lib/utils";
import type { DeploymentNode, RegionMetadata } from "./types";
import { FLAGS } from "./flags";
import { StatusDot } from "./status-dot";

type HealthStatus =
  | "normal"
  | "unstable"
  | "degraded"
  | "unhealthy"
  | "recovering"
  | "health_syncing"
  | "unknown";

type StatusConfig = {
  label: string;
  icon: typeof TriangleWarning2;
  iconColor: string;
  bannerBg: string;
  bannerBorder: string;
  textColor: string;
  message: string;
  statusDotVariant: "success" | "info" | "alert" | "warning" | "error";
  glowShadow: string;
};

const STATUS_CONFIG: Record<HealthStatus, StatusConfig | null> = {
  normal: null, // No banner for normal
  unstable: {
    label: "Unstable",
    icon: TriangleWarning2,
    iconColor: "text-orange-12",
    bannerBg: "bg-orange-2",
    bannerBorder: "border-orangeA-2",
    textColor: "text-orange-12",
    message:
      "Intermittent health check failures. Metrics fluctuating significantly.",
    statusDotVariant: "alert",
    glowShadow:
      "shadow-[inset_0_0_8px_1px_hsl(var(--orangeA-4)),inset_0_0_0_1px_hsl(var(--grayA-5))]",
  },
  degraded: {
    label: "Degraded",
    icon: TriangleWarning,
    iconColor: "text-yellow-12",
    bannerBg: "bg-yellow-2",
    bannerBorder: "border-yellowA-2",
    textColor: "text-yellow-12",
    message: "Performance degraded. Response times elevated.",
    statusDotVariant: "warning",
    glowShadow:
      "shadow-[inset_0_0_8px_1px_hsl(var(--yellowA-4)),inset_0_0_0_1px_hsl(var(--grayA-5))]",
  },
  unhealthy: {
    label: "Unhealthy",
    icon: TriangleWarning2,
    iconColor: "text-red-12",
    bannerBg: "bg-red-2",
    bannerBorder: "border-redA-2",
    textColor: "text-red-12",
    message:
      "Critical health check failures detected. Immediate attention required.",
    statusDotVariant: "error",
    glowShadow:
      "shadow-[inset_0_0_8px_1px_hsl(var(--redA-4)),inset_0_0_0_1px_hsl(var(--grayA-5))]",
  },
  recovering: {
    label: "Recovering",
    icon: CircleCheck, // or a different icon
    iconColor: "text-blue-12",
    bannerBg: "bg-blue-2",
    bannerBorder: "border-blueA-2",
    textColor: "text-blue-12",
    message: "System recovering from issues. Monitoring in progress.",
    statusDotVariant: "info",
    glowShadow:
      "shadow-[inset_0_0_8px_1px_hsl(var(--blueA-4)),inset_0_0_0_1px_hsl(var(--grayA-5))]",
  },
  health_syncing: {
    label: "Syncing",
    icon: Heart, // or a sync icon
    iconColor: "text-info-12",
    bannerBg: "bg-info-2",
    bannerBorder: "border-infoA-2",
    textColor: "text-info-12",
    message: "Health data synchronizing across regions.",
    statusDotVariant: "info",
    glowShadow:
      "shadow-[inset_0_0_8px_1px_hsl(var(--infoA-4)),inset_0_0_0_1px_hsl(var(--grayA-5))]",
  },
  unknown: {
    label: "Unknown",
    icon: TriangleWarning,
    iconColor: "text-gray-11",
    bannerBg: "bg-gray-2",
    bannerBorder: "border-grayA-3",
    textColor: "text-gray-11",
    message: "Unable to determine health status. Investigating.",
    statusDotVariant: "info", // or create a gray variant
    glowShadow:
      "shadow-[inset_0_0_8px_1px_hsl(var(--grayA-4)),inset_0_0_0_1px_hsl(var(--grayA-5))]",
  },
};

type RegionColors = {
  bg: string;
  text: string;
  glow: string;
};

const REGION_COLOR_MAP: Record<string, RegionColors> = {
  "us-east-1": {
    bg: "bg-blueA-2",
    text: "text-blue-10",
    glow: "hover:shadow-[0_4px_16px_-4px_rgba(0,0,0,0.15),0_0_0_1px_hsl(var(--grayA-6)),0_0_30px_color-mix(in_srgb,hsl(var(--blueA-9))_17.5%,transparent)]",
  },
  "ap-east-1": {
    bg: "bg-redA-2",
    text: "text-red-10",
    glow: "hover:shadow-[0_4px_16px_-4px_rgba(0,0,0,0.15),0_0_0_1px_hsl(var(--grayA-6)),0_0_30px_color-mix(in_srgb,hsl(var(--redA-9))_17.5%,transparent)]",
  },
  "ap-south-1": {
    bg: "bg-grassA-2",
    text: "text-grass-10",
    glow: "hover:shadow-[0_4px_16px_-4px_rgba(0,0,0,0.15),0_0_0_1px_hsl(var(--grayA-6)),0_0_30px_color-mix(in_srgb,hsl(var(--grassA-9))_17.5%,transparent)]",
  },
  "eu-west-1": {
    bg: "bg-blueA-2",
    text: "text-blue-10",
    glow: "hover:shadow-[0_4px_16px_-4px_rgba(0,0,0,0.15),0_0_0_1px_hsl(var(--grayA-6)),0_0_30px_color-mix(in_srgb,hsl(var(--blueA-9))_17.5%,transparent)]",
  },
};

const REGION_GRADIENT_MAP: Record<string, string> = {
  "us-east-1": "hsl(var(--infoA-3))",
  "ap-east-1": "hsl(var(--errorA-3))",
  "ap-south-1": "hsl(var(--grassA-3))",
  "eu-west-1": "hsl(var(--infoA-3))",
};

const DEFAULT_COLORS: RegionColors = {
  bg: "bg-grayA-2",
  text: "text-gray-11",
  glow: "hover:shadow-[0_4px_16px_-4px_rgba(0,0,0,0.15),0_0_0_1px_hsl(var(--grayA-6)),0_0_30px_hsl(var(--grayA-3))]",
};

function getRegionFlagCode(
  regionId: string
): RegionMetadata["flagCode"] | undefined {
  switch (true) {
    case regionId.startsWith("us-"):
      return "us";
    case regionId.startsWith("ap-south"):
      return "in";
    case regionId.startsWith("ap-east"):
      return "hk";
    case regionId.startsWith("eu-"):
      return "eu";
    default:
      return undefined;
  }
}

type CardHeaderProps = {
  icon: React.ReactNode;
  title: string;
  subtitle: string;
  gradientColor?: string;
};

function CardHeader({ icon, title, subtitle, gradientColor }: CardHeaderProps) {
  const headerStyle = gradientColor
    ? {
        background: `radial-gradient(circle at 5% 15%, ${gradientColor} 0%, transparent 20%), light-dark(#FFF, #000)`,
      }
    : undefined;

  return (
    <div
      className="border-b border-grayA-4 flex px-3 py-2.5 rounded-t-[14px]"
      style={headerStyle}
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
        <div className="border bg-gray-1 border-grayA-3 h-11 rounded-lg w-8">
          <div className="h-6 border-b border-grayA-3 relative">
            <StatusDot variant="success" />
          </div>
          <div className="h-5 bg-grayA-2 pl-1 pt-[3px]">
            <CircleCheck className="text-gray-9" iconSize="sm-regular" />
          </div>
        </div>
        <div className="border bg-gray-1 border-grayA-3 h-11 rounded-lg w-8 ml-auto">
          <div className="h-6 border-b border-grayA-3 relative">
            <StatusDot variant="info" />
          </div>
          <div className="h-5 bg-grayA-2 pl-1 pt-[3px]">
            <Heart className="text-gray-9" iconSize="sm-regular" />
          </div>
        </div>
      </div>
    </div>
  );
}

type CardFooterProps = {
  flagCode?: RegionMetadata["flagCode"];
  instances?: number;
  power: string | number;
  storage?: string;
};

function CardFooter({ flagCode, instances, power, storage }: CardFooterProps) {
  const FlagComponent = flagCode ? FLAGS[flagCode] : null;

  return (
    <div className="p-1 flex items-center h-full bg-grayA-2 rounded-b-[14px]">
      {FlagComponent && (
        <div className="size-[22px] bg-grayA-3 rounded-full p-[3px] flex items-center justify-center mr-1.5">
          <FlagComponent />
        </div>
      )}
      {instances !== undefined && (
        <div className="bg-grayA-3 p-1.5 flex items-center justify-between rounded-full h-5 gap-1.5">
          <ChartActivity iconSize="sm-medium" className="shrink-0" />
          <span className="text-gray-9 text-[10px] tabular-nums">
            {instances}
          </span>
        </div>
      )}
      <div className="flex items-center gap-2 ml-auto">
        <div className="bg-grayA-3 p-1.5 flex items-center justify-between rounded-full h-5 gap-1.5">
          <Bolt iconSize="sm-medium" className="shrink-0" />
          <span className="text-gray-9 text-[10px] tabular-nums">
            {typeof power === "number" ? `${power}%` : power}
          </span>
        </div>
        {storage && (
          <div className="bg-grayA-3 p-1.5 flex items-center justify-between rounded-full h-5 gap-1.5">
            <Focus iconSize="sm-regular" className="shrink-0" />
            <span className="text-gray-9 text-[10px] tabular-nums">
              {storage}
            </span>
          </div>
        )}
      </div>
    </div>
  );
}

type RegionNodeProps = {
  node: DeploymentNode & { metadata: { type: "region" } };
};

export function RegionNode({ node }: RegionNodeProps) {
  const { flagCode, zones, instances, power, storage } = node.metadata;
  const FlagComponent = FLAGS[flagCode];
  const gradientColor = REGION_GRADIENT_MAP[node.id] ?? "hsl(var(--grayA-3))";
  const colors = REGION_COLOR_MAP[node.id] ?? DEFAULT_COLORS;
  const subtitle = `${zones} availability ${zones > 1 ? "zones" : "zone"}`;

  return (
    <div className="relative w-[282px]">
      {/* Convex top section - higher z-index */}
      <div className="z-10 mx-auto w-[282px] -m-[20px]">
        <div className="h-12  border border-orangeA-2 rounded-t-[14px] bg-orange-2">
          <div className="py-1.5 px-2.5 flex items-center">
            <TriangleWarning2
              className="text-orange-12 shrink-0 mr-2 mb-0.5"
              iconSize="md-regular"
            />
            <span className="text-xs text-orange-12 mr-4 font-medium">
              Unstable
            </span>
            <div className="flex-1 overflow-hidden relative max-w-[200px]">
              <div className="animate-marquee whitespace-nowrap text-orange-12 text-xs">
                Intermittent health check failures. Metrics fluctuating
                significantly.
              </div>
              {/* Left fade - stronger */}
              <div className="absolute left-0 top-0 bottom-0 w-6 bg-gradient-to-r from-orange-2 via-orange-2 to-transparent pointer-events-none" />
              {/* Right fade - stronger */}
              <div className="absolute right-0 top-0 bottom-0 w-6 bg-gradient-to-l from-orange-2 via-orange-2 to-transparent pointer-events-none" />
            </div>
          </div>
        </div>
      </div>

      <div
        className={cn(
          "relative z-20 w-[282px] h-[100px] border border-grayA-4 rounded-[14px] flex flex-col bg-white dark:bg-black shadow-[0_2px_8px_-2px_rgba(0,0,0,0.1)]",
          colors.glow,
          "hover:ring-2 hover:ring-grayA-2 hover:scale-[1.01] transition-all duration-200 cursor-pointer hover:ring-offset-0"
        )}
      >
        <CardHeader
          icon={
            <div className="border rounded-[10px] border-grayA-3 size-9 bg-grayA-3 flex items-center justify-center">
              <FlagComponent />
            </div>
          }
          title={node.label}
          subtitle={subtitle}
          gradientColor={gradientColor}
        />
        <CardFooter instances={instances} power={power} storage={storage} />
      </div>
    </div>
  );
}

type InstanceNodeProps = {
  node: DeploymentNode & { metadata: { type: "instance" } };
  parentRegionId: string;
};

export function InstanceNode({ node, parentRegionId }: InstanceNodeProps) {
  const { description, instances, power, storage } = node.metadata;
  const colors = REGION_COLOR_MAP[parentRegionId] ?? DEFAULT_COLORS;
  const flagCode = getRegionFlagCode(parentRegionId);

  return (
    <div
      className={cn(
        "w-[282px] h-[100px] border border-grayA-4 rounded-[14px] flex flex-col bg-white dark:bg-black shadow-[0_2px_8px_-2px_rgba(0,0,0,0.1)]",
        colors.glow,
        "hover:ring-2 hover:ring-grayA-2 hover:scale-[1.02] transition-all duration-200 cursor-pointer hover:ring-offset-0"
      )}
    >
      <CardHeader
        icon={
          <div
            className={cn(
              "border rounded-[10px] size-9 flex items-center justify-center border-grayA-5",
              colors.bg
            )}
          >
            <Layers3 iconSize="sm-medium" className={colors.text} />
          </div>
        }
        title={node.label}
        subtitle={description}
      />
      <CardFooter
        flagCode={flagCode}
        instances={instances}
        power={power}
        storage={storage}
      />
    </div>
  );
}
