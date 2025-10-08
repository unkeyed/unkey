"use client";
import type { Deployment } from "@/lib/collections";
import { shortenId } from "@/lib/shorten-id";
import {
  InfoTooltip,
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@unkey/ui";
import { format } from "date-fns";
import { PulseIndicator } from "../../details/active-deployment-card/status-indicator";
import { useProject } from "../../layout-provider";

type DeploymentSelectProps = {
  value: string;
  onValueChange: (value: string) => void;
  deployments: Array<{
    deployment: Deployment;
  }>;
  isLoading: boolean;
  placeholder?: string;
  disabled?: boolean;
  id?: string;
  disabledDeploymentId?: string;
};

export function DeploymentSelect({
  value,
  onValueChange,
  deployments,
  isLoading,
  placeholder = "Select deployment",
  disabled = false,
  id,
  disabledDeploymentId,
}: DeploymentSelectProps) {
  const { liveDeploymentId } = useProject();
  const latestDeploymentId = deployments.find(
    ({ deployment }) => deployment.id !== liveDeploymentId,
  )?.deployment.id;

  const getTooltipContent = (
    deploymentId: string,
    isDisabled: boolean,
    isLive: boolean,
    isLatest: boolean,
  ): string | undefined => {
    if (isDisabled) {
      return deploymentId === disabledDeploymentId
        ? "Already selected for comparison"
        : "No OpenAPI spec available";
    }
    if (isLive) {
      return "Live deployment";
    }
    if (isLatest) {
      return "Latest preview deployment";
    }
    return undefined;
  };

  const getTriggerTitle = (): string => {
    if (value === liveDeploymentId) {
      return "Live deployment";
    }
    if (value === latestDeploymentId) {
      return "Latest preview deployment";
    }
    return "";
  };

  const renderOptions = () => {
    if (isLoading) {
      return (
        <SelectItem value="loading" disabled>
          Loading...
        </SelectItem>
      );
    }
    if (deployments.length === 0) {
      return (
        <SelectItem value="no-deployments" disabled>
          No deployments found
        </SelectItem>
      );
    }
    return deployments.map(({ deployment }) => {
      const isDisabled = deployment.id === disabledDeploymentId || !deployment.hasOpenApiSpec;
      const deployedAt = format(deployment.createdAt, "MMM d, h:mm a");
      const isLatest = deployment.id === latestDeploymentId;
      const isLive = deployment.id === liveDeploymentId;
      const tooltipContent = getTooltipContent(deployment.id, isDisabled, isLive, isLatest);

      return (
        <SelectItem key={deployment.id} value={deployment.id} disabled={isDisabled}>
          <InfoTooltip
            disabled={!tooltipContent}
            content={tooltipContent}
            position={{
              side: "right",
              align: "end",
            }}
            asChild
          >
            <div className="flex items-center gap-2.5 text-[13px]">
              <span className="text-grayA-12 font-medium truncate">{shortenId(deployment.id)}</span>
              <span className="text-grayA-9">•</span>
              <span className="text-grayA-9">{deployedAt}</span>
              {isLive && <PulseIndicator />}
              {isLatest && !isLive && (
                <PulseIndicator
                  colors={["bg-gray-9", "bg-gray-7", "bg-gray-8", "bg-gray-9"]}
                  coreColor="bg-gray-9"
                />
              )}
            </div>
          </InfoTooltip>
        </SelectItem>
      );
    });
  };

  return (
    <Select
      value={value}
      onValueChange={onValueChange}
      disabled={disabled || isLoading || deployments.length === 0}
    >
      <SelectTrigger id={id} className="rounded-md" title={getTriggerTitle()}>
        <SelectValue placeholder={placeholder} />
      </SelectTrigger>
      <SelectContent>{renderOptions()}</SelectContent>
    </Select>
  );
}
