"use client";

import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { routes } from "@/lib/navigation/routes";
import { Button } from "@unkey/ui";
import Link from "next/link";
import { IconPlusOutline18 } from "nucleo-ui-outline-18";

export function CreateLogdrainButton() {
  const workspace = useWorkspaceNavigation();

  return (
    <Button
      size="md"
      variant="primary"
      render={<Link href={routes.settings.logdrains.new({ workspaceSlug: workspace.slug })} />}
    >
      <IconPlusOutline18 />
      Create log drain
    </Button>
  );
}
