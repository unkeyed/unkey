import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { useFlag } from "@/lib/flags/provider";
import { Github } from "@unkey/icons";
import { Button, EmptyHero } from "@unkey/ui";
import { useSearchParams } from "next/navigation";
import {
  IconArrowRightOutline18,
  IconBookBookmarkOutline18,
  IconCodeOutline18,
  IconCubeOutline18,
  IconEarthOutline18,
  IconHeartPulseOutline18,
} from "nucleo-ui-outline-18";
import { useState } from "react";
import { CreateProjectDialog } from "../create-project-dialog";
import { DeployPlanGateDialog } from "../deploy-plan-gate-dialog";
import { useDeployGate } from "../hooks/use-deploy-gate";

export function EmptyProjects() {
  const workspace = useWorkspaceNavigation();
  const searchParams = useSearchParams();
  const { gated } = useDeployGate();
  const deployBillingEnabled = useFlag("deployBilling");
  const [isDialogOpen, setIsDialogOpen] = useState(searchParams.get("new") === "true");
  const [isPlanOpen, setIsPlanOpen] = useState(false);

  return (
    <div className="grow w-full flex justify-center items-center p-12">
      <div className="flex flex-col items-center text-center">
        <EmptyHero.Icons className="mb-8">
          <IconEarthOutline18 />
          <Github />
          <IconCubeOutline18 />
          <IconCodeOutline18 />
          <IconHeartPulseOutline18 />
        </EmptyHero.Icons>

        <h2 className="text-accent-12 font-semibold text-2xl leading-8 mb-1">Projects</h2>
        <p className="text-accent-11 text-sm leading-6 max-w-md text-balance mb-6">
          Build, deploy and scale your API inside Unkey. Create a project to get started
          {deployBillingEnabled ? "." : ", free during beta."}
        </p>

        <div className="flex flex-col sm:flex-row items-center justify-center gap-3 w-full">
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
        </div>
      </div>

      <CreateProjectDialog
        isOpen={isDialogOpen}
        onOpenChange={setIsDialogOpen}
        workspaceSlug={workspace.slug}
      />
      <DeployPlanGateDialog isOpen={isPlanOpen} onOpenChange={setIsPlanOpen} from="create" />
    </div>
  );
}
