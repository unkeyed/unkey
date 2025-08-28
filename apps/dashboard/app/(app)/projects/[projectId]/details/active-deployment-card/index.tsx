"use client";
import {
  ChevronDown,
  CircleCheck,
  CircleWarning,
  CircleXMark,
  CodeBranch,
  CodeCommit,
  FolderCloud,
  Layers3,
  Magnifier,
  TriangleWarning2,
} from "@unkey/icons";
import { Badge, Button, Card, CopyButton, Input } from "@unkey/ui";
import { cn } from "@unkey/ui/src/lib/utils";
import { FilterButton } from "./filter-button";
import { useDeploymentLogs } from "./hooks/use-deployment-logs";
import { InfoChip } from "./info-chip";
import { StatusIndicator } from "./status-indicator";

export type DeploymentStatus = "active" | "error" | "pending";

type LogEntry = {
  timestamp: string;
  level?: "info" | "warning" | "error";
  message: string;
};

type DeploymentCardProps = {
  version: string;
  description: string;
  status: DeploymentStatus;
  author: {
    name: string;
    avatar: string;
  };
  createdAt: string;
  branch: string;
  commit: string;
  image: string;
};

const ANIMATION_STYLES = {
  expand: "transition-all duration-400 ease-in",
  slideIn: "transition-all duration-500 ease-out",
} as const;

const STATUS_CONFIG = {
  active: { variant: "success" as const, icon: CircleCheck, text: "Active" },
  error: { variant: "error" as const, icon: CircleWarning, text: "Error" },
  pending: {
    variant: "warning" as const,
    icon: CircleWarning,
    text: "Pending",
  },
} as const;

const MOCK_LOGS: LogEntry[] = [
  {
    timestamp: "13:02:11.007",
    message: "Running build in us-east-1 (Washington, D.C.) — iad1",
  },
  {
    timestamp: "13:02:11.092",
    message: "Cloning github.com/acme/api (Branch: main, Commit: e5f6a7b)",
    level: "error",
  },
  {
    timestamp: "13:02:11.327",
    message: "Build cache not found for this project",
  },
  {
    timestamp: "13:02:11.399",
    message: "Clone completed in 307ms",
    level: "warning",
  },
  { timestamp: "13:02:11.482", message: "Running `unkey build`" },
  { timestamp: "13:02:11.532", message: "Unkey CLI 0.42.1" },
  { timestamp: "13:02:11.590", message: "Validating config files..." },
  { timestamp: "13:02:11.621", message: "✓ env-vars.json validated" },
  { timestamp: "13:02:11.634", message: "✓ runtime.json validated" },
  {
    timestamp: "13:02:11.646",
    message: "✓ secrets.json decrypted successfully",
  },
  {
    timestamp: "13:02:11.657",
    message: "✓ openapi.yaml parsed — 13 endpoints detected",
  },
  {
    timestamp: "13:02:11.665",
    message: '⚠️  Warning: Environment variable "STRIPE_SECRET" is not set. Using fallback value',
    level: "warning",
  },
  { timestamp: "13:02:11.721", message: "Setting up runtime environment" },
  { timestamp: "13:02:11.754", message: "Target image: unkey:latest" },
  {
    timestamp: "13:02:11.903",
    message: "Build environment: nodejs18.x | Linux (x64)",
  },
  { timestamp: "13:02:13.224", message: "Installing dependencies..." },
  { timestamp: "13:02:13.311", message: "✓  Dependencies installed in 1.3s" },
  { timestamp: "13:02:13.642", message: "Compiling project..." },
  { timestamp: "13:02:13.746", message: "✓ Build successful in 331ms" },
  {
    timestamp: "13:02:13.830",
    message: "Registering healthcheck: GET /health every 30s",
  },
  {
    timestamp: "13:02:14.063",
    message: "Checking availability in selected regions...",
  },
  { timestamp: "13:02:14.080", message: "✓ us-east-1 available (2 slots)" },
  { timestamp: "13:02:14.105", message: "✓ eu-west-1 available (1 slot)" },
  { timestamp: "13:02:14.201", message: "✓ ap-south-1 available (1 slot)" },
  { timestamp: "13:02:14.882", message: "Creating deployment image..." },
  {
    timestamp: "13:02:15.014",
    message:
      "❌ Error: Failed to optimize image layer for region eu-west-1. Using fallback strategy",
    level: "error",
  },
  { timestamp: "13:02:15.394", message: "✓ Image built: 210mb" },
  { timestamp: "13:02:15.501", message: "Launching 4 VM instances" },
  {
    timestamp: "13:02:16.212",
    message: "✓ Scaling enabled: 0–5 instances at 80% CPU",
  },
  { timestamp: "13:02:16.501", message: "Deploying to:" },
  { timestamp: "13:02:16.602", message: "  - api.gateway.com (https)" },
  { timestamp: "13:02:16.719", message: "  - internal.api.gateway.com (http)" },
  { timestamp: "13:02:16.801", message: "  - dashboard:3000, 8080, 5792" },
  { timestamp: "13:02:17.023", message: "Activating deployment: v_alpha001" },
  { timestamp: "13:02:17.156", message: "✓ Deployment active" },
  {
    timestamp: "13:02:17.201",
    message: "View logs at /dashboard/logs/alpha001",
  },
  { timestamp: "13:02:17.298", message: "Deployment completed in 5.7s" },
];

export function ActiveDeploymentCard({
  version,
  description,
  status,
  author,
  createdAt,
  branch,
  commit,
  image,
}: DeploymentCardProps) {
  const {
    logFilter,
    searchTerm,
    isExpanded,
    showFade,
    filteredLogs,
    logCounts,
    toggleExpanded,
    handleScroll,
    handleFilterChange,
    handleSearchChange,
    scrollRef,
  } = useDeploymentLogs({ logs: MOCK_LOGS });

  const { variant, icon: StatusIcon, text } = STATUS_CONFIG[status];
  const [imageName, imageTag] = image.split(":");

  return (
    <Card className="rounded-[14px] pt-[14px] flex justify-between flex-col overflow-hidden border-gray-4">
      <div className="flex w-full justify-between items-center px-[22px]">
        <div className="flex gap-5 items-center">
          <StatusIndicator />
          <div className="flex flex-col gap-1">
            <div className="text-accent-12 font-medium text-xs">{version}</div>
            <div className="text-gray-9 text-xs">{description}</div>
          </div>
        </div>
        <div className="flex items-center gap-4">
          <Badge variant={variant} className="text-successA-11 font-medium">
            <div className="flex items-center gap-2">
              <StatusIcon />
              {text}
            </div>
          </Badge>
          <div className="items-center flex gap-2">
            <div className="flex gap-2 items-center">
              <span className="text-gray-9 text-xs">Created by</span>
              <img src={author.avatar} alt={author.name} className="rounded-full size-5" />
              <span className="font-medium text-grayA-12 text-xs">{author.name}</span>
            </div>
          </div>
        </div>
      </div>

      <div className="bg-gray-1 rounded-b-[14px]">
        <div className="relative h-4 flex items-center justify-center">
          <div className="absolute top-0 left-0 right-0 h-4 border-b border-gray-4 rounded-b-[14px] bg-white dark:bg-black" />
        </div>

        <div className="pb-2.5 pt-2 flex justify-between items-center px-3">
          <div className="flex items-center gap-2.5">
            <span className="text-grayA-9 text-xs">{createdAt}</span>
            <div className="flex items-center gap-1.5">
              <InfoChip icon={CodeBranch}>
                <span className="text-grayA-9 text-xs">{branch}</span>
              </InfoChip>
              <InfoChip icon={CodeCommit}>
                <span className="text-grayA-9 text-xs">{commit}</span>
              </InfoChip>
            </div>
            <span className="text-grayA-9 text-xs">using image</span>
            <InfoChip icon={FolderCloud}>
              <div className="text-grayA-10 text-xs">
                <span className="text-gray-12 font-medium">{imageName}</span>:{imageTag}
              </div>
            </InfoChip>
          </div>
          <div className="flex items-center gap-1.5">
            <div className="text-grayA-9 text-xs">Build logs</div>
            <Button size="icon" variant="ghost" onClick={toggleExpanded}>
              <ChevronDown
                className={cn(
                  "text-grayA-9 !size-3 transition-transform duration-200",
                  isExpanded && "rotate-180",
                )}
              />
            </Button>
          </div>
        </div>

        {/* Expandable Logs Section */}
        <div
          className={cn(
            "bg-gray-1 relative overflow-hidden",
            ANIMATION_STYLES.expand,
            isExpanded ? "h-96 opacity-100 py-3" : "h-0 opacity-0 py-0",
          )}
        >
          <div className="flex items-center gap-1.5 px-3 mb-3">
            <FilterButton
              isActive={logFilter === "all"}
              count={logCounts.total}
              onClick={() => handleFilterChange("all")}
              icon={Layers3}
              label="All Logs"
            />
            <FilterButton
              isActive={logFilter === "errors"}
              count={logCounts.errors}
              onClick={() => handleFilterChange("errors")}
              icon={CircleXMark}
              label="Errors"
            />
            <FilterButton
              isActive={logFilter === "warnings"}
              count={logCounts.warnings}
              onClick={() => handleFilterChange("warnings")}
              icon={TriangleWarning2}
              label="Warnings"
            />

            <Input
              variant="ghost"
              wrapperClassName="ml-4"
              className="min-h-[26px] text-xs rounded-lg placeholder:text-grayA-8"
              leftIcon={<Magnifier size="sm-medium" className="text-accent-9 !size-[14px]" />}
              placeholder="Find in logs..."
              value={searchTerm}
              onChange={handleSearchChange}
            />

            <CopyButton
              value={JSON.stringify(filteredLogs)}
              className="size-[22px] [&_svg]:size-[14px] ml-4"
              toastMessage="Logs copied to clipboard"
            />
          </div>

          <div
            className={cn(
              ANIMATION_STYLES.slideIn,
              "h-full",
              isExpanded ? "translate-y-0 opacity-100" : "translate-y-4 opacity-0",
            )}
            style={{
              transitionDelay: isExpanded ? "150ms" : "0ms",
            }}
          >
            <div className="h-full overflow-y-auto" onScroll={handleScroll} ref={scrollRef}>
              {filteredLogs.length === 0 ? (
                <div className="text-center text-gray-9 text-sm py-4 flex items-center justify-center h-full">
                  {searchTerm
                    ? `No logs match "${searchTerm}"`
                    : `No ${logFilter === "all" ? "build" : logFilter} logs available`}
                </div>
              ) : (
                <div className="flex flex-col gap-px">
                  {filteredLogs.map((log, index) => (
                    <div
                      key={`${log.message}-${index}`}
                      className={cn(
                        "font-mono text-xs flex gap-6 items-center text-[11px] leading-7 font-medium",
                        "transition-all duration-300 ease-out",
                        isExpanded ? "translate-x-0 opacity-100" : "translate-x-2 opacity-0",
                        log.level === "warning"
                          ? "bg-warningA-3"
                          : log.level === "error"
                            ? "bg-errorA-5"
                            : "",
                      )}
                      style={{
                        transitionDelay: isExpanded ? `${200 + index * 20}ms` : "0ms",
                      }}
                    >
                      <span className="text-grayA-9 pl-3">{log.timestamp}</span>
                      {log.level && <span className="font-bold">[{log.level.toUpperCase()}]</span>}
                      <span className="text-grayA-12 pr-3">{log.message}</span>
                    </div>
                  ))}
                </div>
              )}
            </div>
          </div>

          {/* Fade overlay */}
          {showFade && (
            <div className="absolute bottom-0 left-0 right-0 h-8 bg-gradient-to-t from-gray-1 to-transparent pointer-events-none transition-opacity duration-300 ease-out" />
          )}
        </div>
      </div>
    </Card>
  );
}
