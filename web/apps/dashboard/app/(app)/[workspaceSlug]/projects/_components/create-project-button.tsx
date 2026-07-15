"use client";

import { routes } from "@/lib/navigation/routes";
import { Plus } from "@unkey/icons";
import { Button } from "@unkey/ui";
import Link from "next/link";

type Props = {
  workspaceSlug: string;
};

export function CreateProjectButton({ workspaceSlug }: Props) {
  return (
    <Button
      size="md"
      variant="primary"
      render={<Link href={routes.projects.new({ workspaceSlug })} />}
    >
      <Plus iconSize="sm-regular" />
      Create project
    </Button>
  );
}
