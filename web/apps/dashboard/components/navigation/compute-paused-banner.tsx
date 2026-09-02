"use client";

import { pausedBody } from "@/app/(app)/[workspaceSlug]/settings/billing/components/compute-paused";
import { formatDollars } from "@/lib/fmt";
import { routes } from "@/lib/navigation/routes";
import { useWorkspace } from "@/providers/workspace-provider";
import Link from "next/link";
import { IconTriangleWarningOutline12 } from "nucleo-ui-outline-12";

/**
 * Workspace-wide banner for the spend-cap paused state. The cap is per
 * workspace and paused means compute is offline everywhere, so this sits at the
 * top of every page (mounted in the app layout) rather than only in billing.
 * Non-dismissible on purpose: it's a rare state that needs action.
 */
export function ComputePausedBanner() {
  const { workspace } = useWorkspace();

  if (!workspace?.deploySpendSuspended) {
    return null;
  }

  const budgetLabel =
    workspace.deploySpendBudgetCents != null
      ? formatDollars(workspace.deploySpendBudgetCents)
      : undefined;

  return (
    <div className="flex h-9 w-full shrink-0 items-center justify-center gap-2 bg-warning-9 px-4 text-center font-medium text-[13px] text-black">
      <IconTriangleWarningOutline12 className="shrink-0" />
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
