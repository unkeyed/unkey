import { TagBadge } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/components/tag-badge";
import { InfoTooltip } from "@unkey/ui";
import type { HTMLAttributes, ReactNode } from "react";

const tooltipContent = {
  enabled: "This environment is enabled and ready to receive deployments.",
  disabled: "This environment is disabled and cannot receive deployments.",
  current: "This environment is currently receiving live traffic.",
  rolledBack: "This environment was previously live but has been rolled back.",
} as const;

type EnvStatusBadgeProps = HTMLAttributes<HTMLDivElement> & {
  variant?: keyof typeof tooltipContent;
  icon?: ReactNode;
  text: string;
};

export const EnvStatusBadge = ({
  variant = "current",
  icon,
  text,
  className,
  ...props
}: EnvStatusBadgeProps) => {
  return (
    <InfoTooltip content={tooltipContent[variant]} variant="inverted">
      <TagBadge variant={variant} icon={icon} text={text} className={className} {...props} />
    </InfoTooltip>
  );
};
