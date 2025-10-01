"use client";
import type { Deployment } from "@/lib/collections";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@unkey/ui";
import { format } from "date-fns";

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
      const isDisabled = deployment.id === disabledDeploymentId;
      const deployedAt = format(deployment.createdAt, "MMM d, h:mm a");
      const commitMessage = deployment.gitCommitMessage?.trim();
      const displayMessage = commitMessage || deployment.gitBranch;
      const truncatedMessage =
        displayMessage.length > 50 ? `${displayMessage.substring(0, 50)}...` : displayMessage;
      const shortSha = deployment.gitCommitSha?.substring(0, 7) || deployment.id.substring(0, 7);

      return (
        <SelectItem
          key={deployment.id}
          value={deployment.id}
          disabled={isDisabled}
          title={isDisabled ? "Already selected for comparison" : undefined}
        >
          <div className="flex items-center gap-2.5 text-[13px]">
            <span className="text-grayA-12 font-medium truncate">{truncatedMessage}</span>
            <span className="text-grayA-9">•</span>
            <span className="text-grayA-9">{deployedAt}</span>
            <span className="text-grayA-9">•</span>
            <span className="text-grayA-9 font-mono text-xs">{shortSha}</span>
          </div>
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
      <SelectTrigger id={id} className="rounded-md">
        <SelectValue placeholder={placeholder} />
      </SelectTrigger>
      <SelectContent>{renderOptions()}</SelectContent>
    </Select>
  );
}
