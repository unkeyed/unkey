"use client";

import { NavbarActionButton } from "@/components/navigation/action-button";
import { Plus } from "@unkey/icons";
import type React from "react";
import { useState } from "react";
import { CreateProjectDialog } from "./create-project-dialog";

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

  return (
    <>
      <NavbarActionButton
        title="Create new project"
        {...rest}
        color="default"
        onClick={() => setIsOpen(true)}
      >
        <Plus />
        Create new project
      </NavbarActionButton>

      <CreateProjectDialog isOpen={isOpen} onOpenChange={setIsOpen} workspaceSlug={workspaceSlug} />
    </>
  );
};
