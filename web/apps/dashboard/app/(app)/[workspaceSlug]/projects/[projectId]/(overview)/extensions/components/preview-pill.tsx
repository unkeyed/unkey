"use client";

import { useEffectiveMode } from "@/lib/extensions/installations";
import type { Extension } from "@/lib/extensions/registry";
import { Badge, InfoTooltip } from "@unkey/ui";
import { cn } from "@unkey/ui/src/lib/utils";

/**
 * Small badge that surfaces when an extension is rendering in preview mode.
 *
 * Returns null for live extensions so it's safe to drop into headers/cards
 * without conditionals at the call site. Matches StatusPill's Badge styling
 * so the two read as siblings instead of competing visual treatments.
 */
export function PreviewPill({ extension }: { extension: Pick<Extension, "mode"> }) {
  const mode = useEffectiveMode(extension);
  if (mode !== "preview") {
    return null;
  }
  const reason =
    extension.mode === "live"
      ? "This extension has a real backend, but the extensionsLive flag is off for this workspace — install is UI-only."
      : "Preview only — install flow is UI-only and isn't wired to a backend yet.";
  return (
    <InfoTooltip content={reason} asChild>
      <Badge variant="secondary" size="sm" className={cn("font-normal")}>
        Preview
      </Badge>
    </InfoTooltip>
  );
}
