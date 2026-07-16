"use client";

import { pausedBody } from "@/app/(app)/[workspaceSlug]/settings/billing/components/compute-paused";
import { formatDollars } from "@/lib/fmt";
import { routes } from "@/lib/navigation/routes";
import { useWorkspace } from "@/providers/workspace-provider";
import { TriangleWarning } from "@unkey/icons";
import Link from "next/link";

/**
 * Height of the paused banner in pixels. The app layout adds this to the fixed
 * sidebar's top offset (via a CSS variable) so nothing renders underneath it.
 */
export const PAUSED_BANNER_HEIGHT = 36;

/**
 * Whether the workspace-wide paused banner should show, with the budget label
 * for its copy. Shared by the banner and the layout so the sidebar's top offset
 * and the banner stay in lockstep.
 */
export function useComputePaused(): { suspended: boolean; budgetLabel?: string } {
  const { workspace } = useWorkspace();
  const suspended = workspace?.deploySpendSuspended ?? false;
  const budgetLabel =
    workspace?.deploySpendBudgetCents != null
      ? formatDollars(workspace.deploySpendBudgetCents)
      : undefined;
  return { suspended, budgetLabel };
}

/**
 * Workspace-wide banner for the spend-cap paused state. The cap is per
 * workspace and paused means compute is offline everywhere, so this sits at the
 * top of every page (mounted in the app layout) rather than only in billing.
 * Non-dismissible on purpose: it's a rare state that needs action. Null unless
 * the workspace is suspended.
 */
export function ComputePausedBanner() {
  const { workspace } = useWorkspace();
  const { suspended, budgetLabel } = useComputePaused();

  if (!workspace || !suspended) {
    return null;
  }

  return (
    <div
      className="flex w-full shrink-0 items-center justify-center gap-2 bg-warning-9 px-4 text-center font-medium text-[13px] text-black"
      style={{ height: PAUSED_BANNER_HEIGHT }}
    >
      <TriangleWarning iconSize="sm-regular" className="shrink-0" />
      <span>
        <span className="font-semibold">Compute paused.</span> {pausedBody(budgetLabel)}{" "}
        <Link
          href={routes.settings.billing({ workspaceSlug: workspace.slug })}
          className="underline underline-offset-2 hover:opacity-80"
        >
          Manage billing
        </Link>
      </span>
    </div>
  );
}
