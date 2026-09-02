"use client";

import { Button } from "@unkey/ui";
import { IconPlusOutline18 } from "nucleo-ui-outline-18";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { routes } from "@/lib/navigation/routes";

export function CreateLogdrainButton({ onClick }: { onClick: () => void }) {
  return (
    <Button size="md" variant="primary" onClick={onClick}>
      <IconPlusOutline18 />
      Create Log Drain
    </Button>
  );
}
