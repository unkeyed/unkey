"use client";

import { Button } from "@unkey/ui";
import { IconPlusOutline18 } from "nucleo-ui-outline-18";

type EnvVarsHeaderProps = {
  isAddOpen: boolean;
  onToggleAdd: () => void;
};

export function EnvVarsHeader({ isAddOpen, onToggleAdd }: EnvVarsHeaderProps) {
  return (
    <div className="flex items-start justify-between">
      <div className="flex flex-col gap-0.5">
        <h1 className="font-medium text-gray-12 text-lg leading-8">Environment Variables</h1>
        <p className="text-[13px] text-gray-11 leading-5">
          Store API keys, tokens, and config securely. Changes apply on next deploy.
        </p>
      </div>
      <Button size="md" onClick={onToggleAdd} variant={isAddOpen ? "outline" : "primary"}>
        <IconPlusOutline18 />
        Add Environment Variable
      </Button>
    </div>
  );
}
