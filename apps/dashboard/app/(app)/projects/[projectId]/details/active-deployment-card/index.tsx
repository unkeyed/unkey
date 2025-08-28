"use client";
import {
  ChevronDown,
  CircleCheck,
  CircleWarning,
  CodeBranch,
  CodeCommit,
  FolderCloud,
} from "@unkey/icons";
import { Badge, Button, Card } from "@unkey/ui";
import { cn } from "@unkey/ui/src/lib/utils";
import { useRef, useState } from "react";
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
  logs?: LogEntry[];
};

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
  logs = MOCK_LOGS,
}: DeploymentCardProps) {
  const [isExpanded, setIsExpanded] = useState(true);
  const [showFade, setShowFade] = useState(true);

  const scrollRef = useRef<HTMLDivElement>(null);

  const statusConfig = {
    active: { variant: "success" as const, icon: CircleCheck, text: "Active" },
    error: { variant: "error" as const, icon: CircleWarning, text: "Error" },
    pending: {
      variant: "warning" as const,
      icon: CircleWarning,
      text: "Pending",
    },
  };

  const { variant, icon: StatusIcon, text } = statusConfig[status];

  const handleToggleLogs = () => {
    setIsExpanded(!isExpanded);
    // Reset scroll position when collapsing
    if (isExpanded) {
      setTimeout(() => {
        if (scrollRef.current) {
          scrollRef.current.scrollTop = 0;
          setShowFade(true);
        }
      }, 50);
    }
  };

  const handleScroll = (e: React.UIEvent<HTMLDivElement>) => {
    const { scrollTop, scrollHeight, clientHeight } = e.currentTarget;
    const isAtBottom = scrollTop + clientHeight >= scrollHeight - 1;
    setShowFade(!isAtBottom);
  };

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
              <div className="gap-2 flex items-center justify-center cursor-pointer border border-grayA-3 transition-all duration-100 bg-grayA-3 p-1.5 h-[22px] rounded-md">
                <CodeBranch size="md-medium" className="text-gray-12" />
                <span className="text-grayA-9 text-xs">{branch}</span>
              </div>
              <div className="gap-2 flex items-center justify-center cursor-pointer border border-grayA-3 transition-all duration-100 bg-grayA-3 p-1.5 h-[22px] rounded-md">
                <CodeCommit size="md-medium" className="text-gray-12" />
                <span className="text-grayA-9 text-xs">{commit}</span>
              </div>
            </div>
            <span className="text-grayA-9 text-xs">using image</span>
            <div className="gap-2 flex items-center justify-center cursor-pointer border border-grayA-3 transition-all duration-100 bg-grayA-3 p-1.5 h-[22px] rounded-md">
              <FolderCloud size="md-medium" className="text-gray-12" />
              <div className="text-grayA-10 text-xs">
                <span className="text-gray-12 font-medium">{image.split(":")[0]}</span>:
                {image.split(":")[1]}
              </div>
            </div>
          </div>
          <div className="flex items-center gap-1.5">
            <div className="text-grayA-9 text-xs">Build logs</div>
            <Button size="icon" variant="ghost" onClick={handleToggleLogs}>
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
            "transition-all duration-400 ease-[cubic-bezier(0.25,0.46,0.45,0.94)]",
            isExpanded ? "h-96 opacity-100 py-3" : "h-0 opacity-0 py-0",
          )}
          style={{
            willChange: isExpanded ? "height, opacity" : "auto",
          }}
        >
          <div
            className={cn(
              "transition-all duration-500 ease-out h-full",
              isExpanded ? "translate-y-0 opacity-100" : "translate-y-4 opacity-0",
            )}
            style={{
              transitionDelay: isExpanded ? "150ms" : "0ms",
            }}
          >
            <div className="h-full overflow-y-auto" onScroll={handleScroll} ref={scrollRef}>
              {logs.length === 0 ? (
                <div className="text-center text-gray-9 text-xs py-4">No build logs available</div>
              ) : (
                <div>
                  {logs.map((log, index) => (
                    <div
                      // biome-ignore lint/suspicious/noArrayIndexKey: <explanation>
                      key={index}
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
                      <span className="text-grayA-9 pl-3 ">{log.timestamp}</span>
                      {log.level === "warning" ? (
                        <span>[WARNING]</span>
                      ) : log.level === "error" ? (
                        <span>[ERROR]</span>
                      ) : null}
                      <span className="text-grayA-12 pr-3">{log.message}</span>
                    </div>
                  ))}
                </div>
              )}
            </div>
          </div>

          {/* Fade overlay - positioned relative to stable container, not animated content */}
          {showFade && (
            <div className="absolute bottom-3 left-0 right-0 h-8 bg-gradient-to-t from-gray-1 to-transparent pointer-events-none transition-opacity duration-300 ease-out" />
          )}
        </div>
      </div>
    </Card>
  );
}
