import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { useFlag } from "@/lib/flags/provider";
import {
  IconArrowRightOutline18,
  IconBookBookmarkOutline18,
  IconCubeOutline18,
} from "@unkey/icons";
import {
  Button,
  EmptyState,
  EmptyStateActions,
  EmptyStateDescription,
  EmptyStateHeader,
  EmptyStateIcon,
  EmptyStateTitle,
} from "@unkey/ui";
import { useState } from "react";
import { CreateProjectDialog } from "../create-project-dialog";
import { DeployPlanGateDialog } from "../deploy-plan-gate-dialog";
import { useDeployGate } from "../hooks/use-deploy-gate";

export function EmptyProjects() {
  const workspace = useWorkspaceNavigation();
  const { gated } = useDeployGate();
  const deployBillingEnabled = useFlag("deployBilling");
  const [isDialogOpen, setIsDialogOpen] = useState(false);
  const [isPlanOpen, setIsPlanOpen] = useState(false);

  return (
    <>
      <EmptyState>
        <EmptyStateIcon>
          <IconCubeOutline18 />
        </EmptyStateIcon>
        <EmptyStateHeader>
          <EmptyStateTitle>Projects</EmptyStateTitle>
          <EmptyStateDescription>
            Build, deploy and scale your API inside Unkey. Create a project to get started
            {deployBillingEnabled ? "." : ", free during beta."}
          </EmptyStateDescription>
        </EmptyStateHeader>
        <EmptyStateActions className="flex-col sm:flex-row justify-center gap-3 w-full">
          <Button
            variant="primary"
            size="md"
            onClick={() => (gated ? setIsPlanOpen(true) : setIsDialogOpen(true))}
            className="w-full max-w-[200px] sm:w-auto sm:max-w-none"
          >
            Create your first project
            <IconArrowRightOutline18 />
          </Button>
          <a
            href="https://www.unkey.com/docs/quickstart/deploy"
            target="_blank"
            rel="noopener noreferrer"
            className="w-full max-w-[200px] sm:w-auto sm:max-w-none"
          >
            <Button variant="outline" size="md" className="w-full sm:w-auto">
              <IconBookBookmarkOutline18 />
              Read the docs
            </Button>
          </a>
        </EmptyStateActions>
      </EmptyState>

      <CreateProjectDialog
        isOpen={isDialogOpen}
        onOpenChange={setIsDialogOpen}
        workspaceSlug={workspace.slug}
      />
      <DeployPlanGateDialog isOpen={isPlanOpen} onOpenChange={setIsPlanOpen} from="create" />
    </>
  );
}
