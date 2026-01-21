import { CircleCheck, CircleWarning } from "@unkey/icons";
import { Badge } from "@unkey/ui";

type DeploymentStatus = "pending" | "building" | "deploying" | "network" | "ready" | "failed";

type StatusConfig = {
  variant: "warning" | "success" | "error";
  icon: React.ComponentType;
  text: string;
};

const STATUS_CONFIG: Record<DeploymentStatus, StatusConfig> = {
  pending: {
    variant: "warning",
    icon: CircleWarning,
    text: "Queued",
  },
  building: {
    variant: "warning",
    icon: CircleWarning,
    text: "Building",
  },
  deploying: {
    variant: "warning",
    icon: CircleWarning,
    text: "Deploying",
  },
  network: {
    variant: "warning",
    icon: CircleWarning,
    text: "Assigning Domains",
  },
  ready: {
    variant: "success",
    icon: CircleCheck,
    text: "Ready",
  },
  failed: {
    variant: "error",
    icon: CircleWarning,
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
  const Icon = config.icon;

  return (
    <Badge variant={config.variant} className={className}>
      <div className="flex items-center gap-2">
        <Icon />
        {config.text}
      </div>
    </Badge>
  );
};
