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
import { useProjectLayout } from "../../layout-provider";

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
  const { liveDeploymentId } = useProjectLayout();
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

      return (
        <SelectItem key={deployment.id} value={deployment.id} disabled={isDisabled}>
          <InfoTooltip
            disabled={!isDisabled && deployment.id !== liveDeploymentId}
            content={
              isDisabled
                ? deployment.id === disabledDeploymentId
                  ? "Already selected for comparison"
                  : "No OpenAPI spec available"
                : deployment.id === liveDeploymentId
                  ? "Live deployment"
                  : undefined
            }
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
              {deployment.id === liveDeploymentId && <PulseIndicator />}
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
      <SelectTrigger
        id={id}
        className="rounded-md"
        title={value === liveDeploymentId ? "Live deployment" : ""}
      >
        <SelectValue placeholder={placeholder} />
      </SelectTrigger>
      <SelectContent>{renderOptions()}</SelectContent>
    </Select>
  );
}
