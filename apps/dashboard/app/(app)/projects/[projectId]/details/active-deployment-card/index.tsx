"use client";

import { trpc } from "@/lib/trpc/client";
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
import { Badge, Button, Card, CopyButton, Input, TimestampInfo } from "@unkey/ui";
import { cn } from "@unkey/ui/src/lib/utils";
import { useProjectLayout } from "../../layout-provider";
import { FilterButton } from "./filter-button";
import { useDeploymentLogs } from "./hooks/use-deployment-logs";
import { InfoChip } from "./info-chip";
import { ActiveDeploymentCardSkeleton } from "./skeleton";
import { StatusIndicator } from "./status-indicator";

const ANIMATION_STYLES = {
  expand: "transition-all duration-400 ease-in",
  slideIn: "transition-all duration-500 ease-out",
} as const;

export const STATUS_CONFIG = {
  success: { variant: "success" as const, icon: CircleCheck, text: "Active" },
  failed: { variant: "error" as const, icon: CircleWarning, text: "Error" },
  pending: {
    variant: "warning" as const,
    icon: CircleWarning,
    text: "Pending",
  },
} as const;

export function ActiveDeploymentCard() {
  const { activeDeploymentId } = useProjectLayout();

  // Get the cached deployment details
  const trpcUtil = trpc.useUtils();
  const deploymentDetails = trpcUtil.deploy.project.details.getData({
    deploymentId: activeDeploymentId,
  });

  const {
    logFilter,
    searchTerm,
    isExpanded,
    showFade,
    filteredLogs,
    logCounts,
    setExpanded,
    handleScroll,
    handleFilterChange,
    handleSearchChange,
    scrollRef,
  } = useDeploymentLogs({ deploymentId: activeDeploymentId });

  if (!deploymentDetails) {
    return <ActiveDeploymentCardSkeleton />;
  }

  const statusConfig = STATUS_CONFIG[deploymentDetails.buildStatus];
  const [imageName, imageTag] = deploymentDetails.image.split(":");

  return (
    <Card className="rounded-[14px] pt-[14px] flex justify-between flex-col overflow-hidden border-gray-4">
      <div className="flex w-full justify-between items-center px-[22px]">
        <div className="flex gap-5 items-center">
          <StatusIndicator />
          <div className="flex flex-col gap-1">
            <div className="text-accent-12 font-medium text-xs">v_alpha001</div>
            <div className="text-gray-9 text-xs">{deploymentDetails.description}</div>
          </div>
        </div>
        <div className="flex items-center gap-4">
          <Badge variant={statusConfig.variant} className="text-successA-11 font-medium">
            <div className="flex items-center gap-2">
              <statusConfig.icon />
              {statusConfig.text}
            </div>
          </Badge>
          <div className="items-center flex gap-2">
            <div className="flex gap-2 items-center">
              <span className="text-gray-9 text-xs">Created by</span>
              <img
                src={deploymentDetails.author.avatar}
                alt={deploymentDetails.author.name}
                className="rounded-full size-5"
              />
              <span className="font-medium text-grayA-12 text-xs">
                {deploymentDetails.author.name}
              </span>
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
            <TimestampInfo
              value={deploymentDetails.createdAt}
              displayType="relative"
              className="text-grayA-9 text-xs"
            />
            <div className="flex items-center gap-1.5">
              <InfoChip icon={CodeBranch}>
                <span className="text-grayA-9 text-xs">{deploymentDetails.branch}</span>
              </InfoChip>
              <InfoChip icon={CodeCommit}>
                <span className="text-grayA-9 text-xs">{deploymentDetails.commit}</span>
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
            <Button size="icon" variant="ghost" onClick={() => setExpanded(!isExpanded)}>
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
                        "transition-all duration-300 ease-out text-grayA-12 ",
                        isExpanded ? "translate-x-0 opacity-100" : "translate-x-2 opacity-0",
                        log.level === "warning"
                          ? "bg-gradient-to-r from-warningA-3 to-warningA-1 text-warningA-12"
                          : log.level === "error"
                            ? "bg-gradient-to-r from-errorA-3 to-errorA-1 text-errorA-12"
                            : "",
                      )}
                      style={{
                        transitionDelay: isExpanded ? `${200 + index * 20}ms` : "0ms",
                      }}
                    >
                      <span className="text-grayA-9 pl-3">{log.timestamp}</span>
                      {log.level && (
                        <span className="font-medium">[{log.level.toUpperCase()}]</span>
                      )}
                      <span className="pr-3">{log.message}</span>
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
