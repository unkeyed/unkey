"use client";

import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { routes } from "@/lib/navigation/routes";
import { Plus } from "@unkey/icons";
import { Button } from "@unkey/ui";
import Link from "next/link";

export function CreateLogdrainButton() {
  const workspace = useWorkspaceNavigation();

  return (
    <Button
      size="md"
      variant="primary"
      render={<Link href={routes.settings.logdrains.new({ workspaceSlug: workspace.slug })} />}
    >
      <Plus iconSize="sm-regular" />
      Create log drain
    </Button>
  );
}
