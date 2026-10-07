"use client";

import { IconPlusOutline18 } from "@unkey/icons";
import { Button } from "@unkey/ui";

type EnvVarsHeaderProps = {
  isAddOpen: boolean;
  onToggleAdd: () => void;
};

export function EnvVarsHeader({ isAddOpen, onToggleAdd }: EnvVarsHeaderProps) {
  return (
    <div className="flex items-start justify-between">
      <h1 className="font-semibold text-gray-12 text-lg leading-8">Environment variables</h1>
      <Button size="md" onClick={onToggleAdd} variant={isAddOpen ? "outline" : "primary"}>
        <IconPlusOutline18 />
        Add Environment Variable
      </Button>
    </div>
  );
}
