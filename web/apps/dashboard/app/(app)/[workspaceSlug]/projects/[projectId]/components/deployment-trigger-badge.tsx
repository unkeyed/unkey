import { BracketsCurly, Github, Laptop2, SquareTerminal, Unkey } from "@unkey/icons";
import { InfoTooltip } from "@unkey/ui";
import type { ReactNode } from "react";

type DeploymentTrigger = "unknown" | "github" | "api" | "cli" | "dashboard" | "unkey";

type DeploymentTriggerBadgeProps = {
  trigger: DeploymentTrigger;
  triggeredBy?: string | null;
  triggerReason?: string | null;
};

type Spec = { label: string; icon: ReactNode };

const SPECS: Record<Exclude<DeploymentTrigger, "unknown">, Spec> = {
  github: { label: "GitHub", icon: <Github iconSize="sm-regular" /> },
  dashboard: { label: "Dashboard", icon: <Laptop2 iconSize="sm-regular" /> },
  api: { label: "API", icon: <BracketsCurly iconSize="sm-regular" /> },
  cli: { label: "CLI", icon: <SquareTerminal iconSize="sm-regular" /> },
  unkey: { label: "Unkey", icon: <Unkey iconSize="sm-regular" /> },
};

// Prefix `triggered_by` into a readable tooltip line, per trigger semantics.
function formatActor(trigger: DeploymentTrigger, triggeredBy?: string | null): string | null {
  if (!triggeredBy) {
    return null;
  }
  switch (trigger) {
    case "github":
      return `Pushed by @${triggeredBy}`;
    case "dashboard":
    case "unkey":
      return `By user ${triggeredBy}`;
    case "api":
    case "cli":
      return `By root key ${triggeredBy}`;
    default:
      return triggeredBy;
  }
}

export function DeploymentTriggerBadge({
  trigger,
  triggeredBy,
  triggerReason,
}: DeploymentTriggerBadgeProps) {
  // Historical rows before this column existed have trigger="unknown";
  // hide rather than surface a noisy "Unknown" label on every old row.
  if (trigger === "unknown") {
    return null;
  }

  const spec = SPECS[trigger];

  // Unkey-triggered deployments get a distinct color so customers notice
  // when we touched their deployment. Others stay in neutral tones to
  // match the surrounding MetadataCell styling (CodeBranch etc).
  const tone = trigger === "unkey" ? "text-warningA-11" : "text-accent-12";

  const content = (
    <span className={`inline-flex items-center gap-1.5 text-xs ${tone}`}>
      <span className="shrink-0">{spec.icon}</span>
      <span className="font-medium">{spec.label}</span>
    </span>
  );

  const actor = formatActor(trigger, triggeredBy);
  const tooltipParts = [actor, triggerReason].filter(Boolean);
  if (tooltipParts.length === 0) {
    return content;
  }

  return (
    <InfoTooltip content={tooltipParts.join(" — ")} asChild position={{ side: "top" }}>
      {content}
    </InfoTooltip>
  );
}
