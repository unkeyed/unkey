import { CircleWarning } from "@unkey/icons";
import { Badge } from "@unkey/ui";
import { cn } from "@unkey/ui/src/lib/utils";

type DeploymentStatus = "pending" | "building" | "deploying" | "network" | "ready" | "failed";

type StatusConfig = {
  variant: "warning" | "success" | "error" | "secondary";
  text: string;
};

const STATUS_CONFIG: Record<DeploymentStatus, StatusConfig> = {
  pending: {
    variant: "secondary",
    text: "Queued",
  },
  building: {
    variant: "secondary",
    text: "Building",
  },
  deploying: {
    variant: "secondary",
    text: "Deploying",
  },
  network: {
    variant: "secondary",
    text: "Assigning Domains",
  },
  ready: {
    variant: "success",
    text: "Ready",
  },
  failed: {
    variant: "error",
    text: "Error",
  },
};

type Props = {
  status?: DeploymentStatus;
  className?: string;
};

export const DeploymentStatusBadge = ({ status, className }: Props) => {
  if (!status) {
    throw new Error(`Invalid deployment status: ${status}`);
  }

  const config = STATUS_CONFIG[status];

  return (
    <Badge variant={config.variant} className={cn("font-medium", className)}>
      <div className="flex items-center gap-2">
        <CircleWarning iconSize="md-regular" />
        {config.text}
      </div>
    </Badge>
  );
};
