"use client";
import { WorkspaceSwitcher } from "@/components/navigation/sidebar/team-switcher";
import { WorkspaceSwitcherTanStack } from "@/components/navigation/sidebar/workspace-switcher-tanstack";
import { Button } from "@unkey/ui";
import type React from "react";
import { useState } from "react";

type Props = {
  workspace: {
    name: string;
  };
};

export const WorkspaceSwitcherWrapper: React.FC<Props> = (props): JSX.Element => {
  // Toggle to test TanStack DB implementation
  const [useTanStack, setUseTanStack] = useState(false);

  return (
    <div className="flex flex-col gap-2 w-full">
      {useTanStack ? (
        <WorkspaceSwitcherTanStack workspace={props.workspace} />
      ) : (
        <WorkspaceSwitcher workspace={props.workspace} />
      )}

      {/* Toggle button - only show when sidebar is expanded */}
      <Button
        variant="outline"
        size="sm"
        className="text-xs h-6"
        onClick={() => setUseTanStack(!useTanStack)}
      >
        {useTanStack ? "TanStack DB" : "tRPC"}
      </Button>
    </div>
  );
};
