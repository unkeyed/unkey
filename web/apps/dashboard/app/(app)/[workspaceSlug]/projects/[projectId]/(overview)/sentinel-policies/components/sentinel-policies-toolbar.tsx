"use client";

import { ChevronDown, Layers3 } from "@unkey/icons";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@unkey/ui";

type SentinelPoliciesToolbarProps = {
  environmentId: string;
  onEnvironmentChange: (value: string) => void;
  environments: { id: string; slug: string }[];
};

export function SentinelPoliciesToolbar({
  environmentId,
  onEnvironmentChange,
  environments,
}: SentinelPoliciesToolbarProps) {
  return (
    <div className="flex items-stretch ml-auto">
      <div className="max-w-[220px]">
        <Select value={environmentId} onValueChange={onEnvironmentChange}>
          <SelectTrigger
            className="h-9 w-full bg-gray-1 capitalize"
            leftIcon={<Layers3 iconSize="md-medium" className="text-gray-9" />}
            rightIcon={<ChevronDown className="absolute right-2" iconSize="md-medium" />}
          >
            <SelectValue placeholder="Select Environment" />
          </SelectTrigger>
          <SelectContent>
            {environments.map((env) => (
              <SelectItem key={env.id} value={env.id} className="capitalize">
                {env.slug}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>
    </div>
  );
}
