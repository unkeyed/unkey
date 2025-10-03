"use client";

import { formatNumber } from "@/lib/fmt";
import { eq, useLiveQuery } from "@tanstack/react-db";
import {
  ChevronDown,
  CircleCheck,
  CircleWarning,
  CircleXMark,
  CodeBranch,
  CodeCommit,
  Layers3,
  Magnifier,
  TriangleWarning2,
} from "@unkey/icons";
import { Badge, Button, CopyButton, Input, TimestampInfo } from "@unkey/ui";
import { cn } from "@unkey/ui/src/lib/utils";
import { format } from "date-fns";
import { useProject } from "../../layout-provider";
import { Card } from "../card";
import { ActiveDeploymentCardEmpty } from "./active-deployment-card-empty";
import { FilterButton } from "./filter-button";
import { Avatar } from "./git-avatar";
import { useDeploymentLogs } from "./hooks/use-deployment-logs";
import { InfoChip } from "./info-chip";
import { ActiveDeploymentCardSkeleton } from "./skeleton";
import { StatusIndicator } from "./status-indicator";

const ANIMATION_STYLES = {
  expand: "transition-all duration-400 ease-in",
  slideIn: "transition-all duration-500 ease-out",
} as const;

export const statusIndicator = (
  status: "pending" | "building" | "deploying" | "network" | "ready" | "failed",
) => {
  switch (status) {
    case "pending":
      return {
        variant: "warning" as const,
        icon: CircleWarning,
        text: "Queued",
      };
    case "building":
      return {
        variant: "warning" as const,
        icon: CircleWarning,
        text: "Building",
      };
    case "deploying":
      return {
        variant: "warning" as const,
        icon: CircleWarning,
        text: "Deploying",
      };
    case "network":
      return {
        variant: "warning" as const,
        icon: CircleWarning,
        text: "Assigning Domains",
      };
    case "ready":
      return { variant: "success" as const, icon: CircleCheck, text: "Ready" };
    case "failed":
      return { variant: "error" as const, icon: CircleWarning, text: "Error" };
  }

  return { variant: "error" as const, icon: CircleWarning, text: "Unknown" };
};

type Props = {
  deploymentId: string | null;
};

export const ActiveDeploymentCard = ({ deploymentId }: Props) => {
  const { collections } = useProject();
  const { data, isLoading } = useLiveQuery(
    (q) =>
      q
        .from({ deployment: collections.deployments })
        .where(({ deployment }) => eq(deployment.id, deploymentId)),
    [deploymentId],
  );
  const deployment = data.at(0);

  // If deployment status is not ready it means we gotta keep showing build steps.
  // Then, user can switch between runtime(not implemented yet) and gateway logs
  const showBuildSteps = deployment?.status !== "ready";

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
  } = useDeploymentLogs({
    deploymentId,
    showBuildSteps,
  });

  if (isLoading) {
    return <ActiveDeploymentCardSkeleton />;
  }
  if (!deployment) {
    return <ActiveDeploymentCardEmpty />;
  }

  const statusConfig = statusIndicator(deployment.status);

  return (
    <Card className="rounded-[14px] pt-[14px] flex justify-between flex-col overflow-hidden border-gray-4">
      <div className="flex w-full justify-between items-center px-[22px]">
        <div className="flex gap-5 items-center">
          <StatusIndicator withSignal />
          <div className="flex flex-col gap-1">
            <div className="text-accent-12 font-medium text-xs">{deployment.id}</div>
            <div className="text-gray-9 text-xs">{deployment.gitCommitMessage}</div>
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
              <Avatar src={deployment.gitCommitAuthorAvatarUrl} alt="Author" />
              <span className="font-medium text-grayA-12 text-xs">
                {deployment.gitCommitAuthorHandle}
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
              value={deployment.createdAt}
              displayType="relative"
              className="text-grayA-9 text-xs"
            />
            <div className="flex items-center gap-1.5">
              <InfoChip icon={CodeBranch}>
                <span className="text-grayA-9 text-xs truncate max-w-32">
                  {deployment.gitBranch}
                </span>
              </InfoChip>
              <InfoChip icon={CodeCommit}>
                <span className="text-grayA-9 text-xs">
                  {(deployment.gitCommitSha ?? "").slice(0, 7)}
                </span>
              </InfoChip>
            </div>
          </div>
          <div className="flex items-center gap-1.5">
            <div className="text-grayA-9 text-xs">
              {showBuildSteps ? "Build logs" : "Gateway logs"}
            </div>
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
              count={formatNumber(logCounts.total)}
              onClick={() => handleFilterChange("all")}
              icon={Layers3}
              label="All Logs"
            />
            {/*//INFO: Let's keep them for now we might need them in the future*/}
            <FilterButton
              isActive={logFilter === "errors"}
              count={formatNumber(logCounts.errors)}
              onClick={() => handleFilterChange("errors")}
              icon={CircleXMark}
              label="Errors"
            />
            <FilterButton
              isActive={logFilter === "warnings"}
              count={formatNumber(logCounts.warnings)}
              onClick={() => handleFilterChange("warnings")}
              icon={TriangleWarning2}
              label="Warnings"
            />

            <Input
              variant="ghost"
              wrapperClassName="ml-4"
              className="min-h-[26px] text-xs rounded-lg placeholder:text-grayA-8"
              leftIcon={<Magnifier iconsize="sm-medium" className="text-accent-9 !size-[14px]" />}
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
                    : `No ${
                        logFilter === "all" ? (showBuildSteps ? "build" : "gateway") : logFilter
                      } logs available`}
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
                      <span className="text-grayA-9 pl-3">
                        {format(new Date(log.timestamp), "HH:mm:ss.SSS")}
                      </span>
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
};
