"use client";

import { routes } from "@/lib/navigation/routes";
import { Plus } from "@unkey/icons";
import { Button } from "@unkey/ui";
import Link from "next/link";

type Props = {
  workspaceSlug: string;
  variant?: "primary" | "outline";
};

export function CreateProjectButton({ workspaceSlug, variant = "primary" }: Props) {
  return (
    <Button
      size="md"
      variant={variant}
      render={<Link href={routes.projects.new({ workspaceSlug })} />}
    >
      <Plus iconSize="sm-regular" />
      Create project
    </Button>
  );
}
