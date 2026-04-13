"use client";

import { Plus, Sparkle3 } from "@unkey/icons";
import { Button } from "@unkey/ui";

type SentinelPoliciesHeaderProps = {
  onAddPolicy: () => void;
  onGenerateWithAi: () => void;
};

export function SentinelPoliciesHeader({
  onAddPolicy,
  onGenerateWithAi,
}: SentinelPoliciesHeaderProps) {
  return (
    <div className="flex items-start justify-between">
      <div className="flex flex-col gap-0.5">
        <h1 className="font-semibold text-gray-12 text-lg leading-8">Sentinel Policies</h1>
        <p className="text-[13px] text-gray-11 leading-5">
          Middleware policy chains that protect your API. Policies are evaluated in order, drag to
          reorder.
        </p>
      </div>
      <div className="flex items-center gap-2">
        <Button size="md" onClick={onGenerateWithAi} variant="outline">
          <Sparkle3 iconSize="sm-regular" />
          Generate with AI
        </Button>
        <Button size="md" onClick={onAddPolicy} variant="primary">
          <Plus iconSize="sm-regular" />
          Add Policy
        </Button>
      </div>
    </div>
  );
}
