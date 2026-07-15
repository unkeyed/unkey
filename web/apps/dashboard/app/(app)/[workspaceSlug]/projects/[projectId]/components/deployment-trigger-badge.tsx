import { BracketsCurly, Github, Laptop2, SquareTerminal, Unkey } from "@unkey/icons";
import { InfoTooltip } from "@unkey/ui";
import type { ReactNode } from "react";

type DeploymentTrigger = "unknown" | "github" | "api" | "cli" | "dashboard" | "unkey";

type DeploymentTriggerBadgeProps = {
  trigger: DeploymentTrigger;
  triggeredBy?: string | null;
  triggerReason?: string | null;
  // When true, render only the icon (with tooltip carrying the label
  // and actor info). Used in dense surfaces like the deployments list
  // row where another bordered badge already sits next to it.
  iconOnly?: boolean;
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
  iconOnly = false,
}: DeploymentTriggerBadgeProps) {
  // Historical rows before this column existed have trigger="unknown";
  // hide rather than surface a noisy "Unknown" label on every old row.
  if (trigger === "unknown") {
    return null;
  }

  const spec = SPECS[trigger];

  // Unkey-triggered deployments get a distinct color so customers notice
  // when we touched their deployment. Others render in neutral gray so
  // they sit quietly next to status / git-branch metadata without
  // competing with the primary status badge.
  const isUnkey = trigger === "unkey";

  const content = (
    <span
      className={`inline-flex w-fit items-center rounded-md text-xs font-medium leading-none ${iconOnly ? "h-5.5 w-5.5 justify-center p-1" : "h-5.5 gap-1.5 px-1.5 py-1"} ${isUnkey ? "bg-warningA-3 text-warningA-11" : "bg-grayA-3 text-gray-12"}`}
      aria-label={iconOnly ? `Triggered via ${spec.label}` : undefined}
    >
      <span className="shrink-0">{spec.icon}</span>
      {iconOnly ? null : <span>{spec.label}</span>}
    </span>
  );

  const actor = formatActor(trigger, triggeredBy);
  // In iconOnly mode the label has to live in the tooltip too,
  // otherwise hovering an unfamiliar icon yields nothing.
  const tooltipParts = [iconOnly ? `Triggered via ${spec.label}` : null, actor, triggerReason]
    .filter(Boolean)
    .join(" — ");

  if (!tooltipParts) {
    return content;
  }

  return (
    <InfoTooltip content={tooltipParts} asChild position={{ side: "top" }}>
      {content}
    </InfoTooltip>
  );
}
