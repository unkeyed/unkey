"use client";

import { Button } from "@unkey/ui";
import { IconPlusOutline18 } from "nucleo-ui-outline-18";
import { useState } from "react";
import { CreateProjectDialog } from "./create-project-dialog";
import { DeployPlanGateDialog } from "./deploy-plan-gate-dialog";
import { useDeployGate } from "./hooks/use-deploy-gate";

type Props = {
  defaultOpen?: boolean;
  workspaceSlug: string;
};

export function CreateProjectButton({ defaultOpen, workspaceSlug }: Props) {
  const [isOpen, setIsOpen] = useState(defaultOpen ?? false);
  const [isPlanOpen, setIsPlanOpen] = useState(false);

  const { gated, isLoading } = useDeployGate();

  return (
    <>
      <Button
        size="md"
        variant="primary"
        loading={isLoading}
        onClick={() => (gated ? setIsPlanOpen(true) : setIsOpen(true))}
      >
        <IconPlusOutline18 />
        Create project
      </Button>

      <CreateProjectDialog isOpen={isOpen} onOpenChange={setIsOpen} workspaceSlug={workspaceSlug} />
      <DeployPlanGateDialog isOpen={isPlanOpen} onOpenChange={setIsPlanOpen} from="create" />
    </>
  );
}
