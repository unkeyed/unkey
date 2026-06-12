"use client";

import { NavbarActionButton } from "@/components/navigation/action-button";
import { Plus } from "@unkey/icons";
import { InfoTooltip } from "@unkey/ui";
import type React from "react";
import { useState } from "react";
import { CreateProjectDialog } from "./create-project-dialog";
import { useDeployGate } from "./hooks/use-deploy-gate";

type Props = {
  defaultOpen?: boolean;
  workspaceSlug: string;
};

export const CreateProjectButton = ({
  defaultOpen,
  workspaceSlug,
  ...rest
}: React.ButtonHTMLAttributes<HTMLButtonElement> & Props) => {
  const [isOpen, setIsOpen] = useState(defaultOpen ?? false);

  // When gated (deployBilling on + no Deploy entitlement), disable creation so
  // the button is never a dead-end click; the projects screen carries the
  // "Choose a plan" paywall. The authoritative gate is in ctrl-api.
  const { gated } = useDeployGate();

  return (
    <>
      <InfoTooltip
        content="A Compute plan is required to create projects."
        disabled={!gated}
        asChild
      >
        <NavbarActionButton
          title="Create new project"
          {...rest}
          color="default"
          disabled={gated}
          onClick={() => setIsOpen(true)}
        >
          <Plus />
          Create new project
        </NavbarActionButton>
      </InfoTooltip>

      <CreateProjectDialog isOpen={isOpen} onOpenChange={setIsOpen} workspaceSlug={workspaceSlug} />
    </>
  );
};
