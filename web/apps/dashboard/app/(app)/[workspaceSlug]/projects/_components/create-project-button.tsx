"use client";

import { Plus } from "@unkey/icons";
import { Button } from "@unkey/ui";
import { useState } from "react";
import { CreateProjectDialog } from "./create-project-dialog";

type Props = {
  defaultOpen?: boolean;
  workspaceSlug: string;
};

export function CreateProjectButton({ defaultOpen, workspaceSlug }: Props) {
  const [isOpen, setIsOpen] = useState(defaultOpen ?? false);

  return (
    <>
      <Button size="md" variant="primary" onClick={() => setIsOpen(true)}>
        <Plus iconSize="sm-regular" />
        Create project
      </Button>

      <CreateProjectDialog isOpen={isOpen} onOpenChange={setIsOpen} workspaceSlug={workspaceSlug} />
    </>
  );
}
