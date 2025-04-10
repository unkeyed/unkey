import type { KeyDetails } from "@/lib/trpc/routers/api/keys/query-api-keys/schema";
import { cn } from "@/lib/utils";
import {
  Ban,
  CircleCaretRight,
  CircleCheck,
  CircleHalfDottedClock,
  ShieldKey,
  TriangleWarning2,
} from "@unkey/icons";
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "@unkey/ui";
import { useFetchVerificationTimeseries } from "../bar-chart/use-fetch-timeseries";

type StatusType =
  | "disabled"
  | "low-credits"
  | "expires-soon"
  | "rate-limited"
  | "validation-issues"
  | "operational";

type StatusInfo = {
  type: StatusType;
  label: string;
  color: string;
  icon: string;
  tooltip: string;
  priority: number;
};

interface StatusBadgeProps {
  primary: Omit<StatusInfo, "priority" | "tooltip">;
  count: number;
}

interface StatusDisplayProps {
  keyData: KeyDetails;
  keyAuthId: string;
}

const StatusBadge = ({ primary, count }: StatusBadgeProps) => {
  const IconMap: Record<string, React.ReactNode> = {
    Clock: <CircleHalfDottedClock size="sm-regular" />,
    Coins: <CircleCaretRight size="sm-regular" />,
    AlertTriangle: <TriangleWarning2 size="sm-regular" />,
    CheckCircle: <CircleCheck size="sm-regular" />,
    Ban: <Ban size="sm-regular" />,
    ShieldAlert: <ShieldKey size="sm-regular" />,
  };
  // Choose the correct icon component
  const IconComponent = primary.icon ? IconMap[primary.icon] : null;
  return (
    <div className="flex items-center justify-start gap-0.5">
      <div
        className={cn(
          primary.color,
          "px-1.5 py-1 flex items-center justify-center gap-2 h-[22px]",
          count > 0 ? "rounded-l" : "rounded-md",
        )}
      >
        {IconComponent && <span>{IconComponent}</span>}
        <span>{primary.label}</span>
      </div>
      {count > 0 && (
        <div
          className={cn(
            primary.color,
            "rounded-r px-1.5 py-1 flex items-center justify-center h-[22px]",
          )}
        >
          +{count}
        </div>
      )}
    </div>
  );
};

export const StatusDisplay: React.FC<StatusDisplayProps> = ({ keyAuthId, keyData }) => {
  // Fetch validation outcomes data using the existing hook
  const { timeseries, isError, isLoading } = useFetchVerificationTimeseries(keyAuthId, keyData.id);

  // Determine which status to show based on priority
  const getStatusInfo = () => {
    // Track all applicable statuses
    const statuses: StatusInfo[] = [];

    // If key is disabled, that's the only status we show
    if (!keyData.enabled) {
      return {
        primary: {
          label: "Disabled",
          color: "bg-grayA-3 text-grayA-11",
          icon: "Ban",
        },
        count: 0,
        tooltips: ["This key is currently disabled and cannot be used for verification."],
      };
    }

    // Aggregate data from all timeseries points
    if (timeseries.length > 0) {
      // Calculate totals across all data points
      const aggregatedData = timeseries.reduce(
        (acc, point) => {
          acc.valid += point.valid;
          acc.total += point.total;
          acc.error += point.error;

          // Sum up specific error types
          if (point.rate_limited) {
            acc.rate_limited += point.rate_limited;
          }
          if (point.insufficient_permissions) {
            acc.insufficient_permissions += point.insufficient_permissions;
          }
          if (point.forbidden) {
            acc.forbidden += point.forbidden;
          }
          if (point.disabled) {
            acc.disabled += point.disabled;
          }
          if (point.expired) {
            acc.expired += point.expired;
          }
          if (point.usage_exceeded) {
            acc.usage_exceeded += point.usage_exceeded;
          }

          return acc;
        },
        {
          valid: 0,
          total: 0,
          error: 0,
          rate_limited: 0,
          insufficient_permissions: 0,
          forbidden: 0,
          disabled: 0,
          expired: 0,
          usage_exceeded: 0,
        },
      );

      const total = aggregatedData.total;

      // Check for rate limiting issues (more than 10% rate limited)
      if (
        aggregatedData.rate_limited > 0 &&
        total > 0 &&
        aggregatedData.rate_limited / total > 0.1
      ) {
        statuses.push({
          type: "rate-limited",
          label: "Ratelimited",
          color: "bg-errorA-3 text-errorA-11",
          icon: "AlertTriangle",
          tooltip:
            "This key is getting ratelimited frequently. Check the configured ratelimits and reach out to your user about their usage.",
          priority: 2,
        });
      }

      // Check for general validation issues (more than 10% errors)
      if (total > 0 && aggregatedData.error / total > 0.1) {
        statuses.push({
          type: "validation-issues",
          label: "Potential issues",
          color: "bg-warningA-3 text-warningA-11",
          icon: "ShieldAlert",
          tooltip:
            "This key has a high error rate. Please check its logs to debug potential issues.",
          priority: 3,
        });
      }
    }

    // Check for low credits
    if (
      // Less than 100 credits remaining
      (keyData.key.remaining != null && keyData.key.remaining < 100) ||
      // OR less than 10% of automatic refill amount
      (keyData.key.refillAmount &&
        keyData.key.remaining != null &&
        keyData.key.refillAmount > 0 &&
        keyData.key.remaining < keyData.key.refillAmount / 10)
    ) {
      statuses.push({
        type: "low-credits",
        label: "Low Credits",
        color: "bg-errorA-3 text-errorA-11",
        icon: "Coins",
        tooltip: "This key has a low credit balance. Top it off to prevent disruptions.",
        priority: 1, // Highest priority
      });
    }

    // Check for expiring soon (within 24 hours)
    if (keyData.expires) {
      const expiry = keyData.expires;
      const hoursToExpiry = (expiry - Date.now() / 1000) / (60 * 60);
      if (hoursToExpiry > 0 && hoursToExpiry <= 24) {
        statuses.push({
          type: "expires-soon",
          label: "Expires soon",
          color: "bg-warningA-3 text-warningA-11",
          icon: "Clock",
          tooltip:
            "This key will expire in less than 24 hours. Rotate the key or extend its deadline to prevent disruptions.",
          priority: 2,
        });
      }
    }

    // If no issues, show operational
    if (statuses.length === 0) {
      return {
        primary: {
          label: "Operational",
          color: "bg-successA-3 text-successA-11",
          icon: "CheckCircle",
        },
        count: 0,
        tooltips: ["This key is operating normally."],
      };
    }

    // Sort by priority (lower number = higher priority)
    statuses.sort((a, b) => a.priority - b.priority);

    // Return the highest priority status and total count of issues
    return {
      primary: {
        label: statuses[0].label,
        color: statuses[0].color,
        icon: statuses[0].icon,
      },
      count: statuses.length - 1, // Don't count the primary one in the "+X" indicator
      tooltips: statuses.map((s) => s.tooltip),
    };
  };

  const statusInfo = getStatusInfo();

  return (
    <TooltipProvider>
      <Tooltip>
        <TooltipTrigger asChild>
          <div>
            <StatusBadge primary={statusInfo.primary} count={statusInfo.count} />
          </div>
        </TooltipTrigger>
        <TooltipContent>
          {statusInfo.tooltips?.map((tooltip, i) => (
            <p key={i} className={i > 0 ? "mt-2" : ""}>
              {tooltip}
            </p>
          ))}
        </TooltipContent>
      </Tooltip>
    </TooltipProvider>
  );
};
