"use client";

import { Plus } from "@unkey/icons";
import { Button, InfoTooltip } from "@unkey/ui";
import { useState } from "react";
import { CreateProjectDialog } from "./create-project-dialog";
import { useDeployGate } from "./hooks/use-deploy-gate";

type Props = {
  defaultOpen?: boolean;
  workspaceSlug: string;
};

export function CreateProjectButton({ defaultOpen, workspaceSlug }: Props) {
  const [isOpen, setIsOpen] = useState(defaultOpen ?? false);

  // UX-only mirror of the authoritative ctrl-api gate, so a gated user gets a
  // disabled button into the paywall instead of a request that fails.
  const { gated, isLoading } = useDeployGate();

  return (
    <>
      <InfoTooltip
        content="A Compute plan is required to create projects."
        disabled={!gated}
        asChild
      >
        <span>
          <Button
            size="md"
            variant="primary"
            loading={isLoading}
            disabled={gated}
            onClick={() => setIsOpen(true)}
          >
            <Plus iconSize="sm-regular" />
            Create project
          </Button>
        </span>
      </InfoTooltip>

      <CreateProjectDialog isOpen={isOpen} onOpenChange={setIsOpen} workspaceSlug={workspaceSlug} />
    </>
  );
}
