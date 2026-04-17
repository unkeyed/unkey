"use client";

import { ChevronDown, Layers3 } from "@unkey/icons";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@unkey/ui";
import { useProjectData } from "../../../../data-provider";
import { useFilters } from "../../../hooks/use-filters";

export function EnvironmentSelect() {
  const { environments } = useProjectData();
  const { filters, updateFilters } = useFilters();

  const activeEnvFilter = filters.find((f) => f.field === "environment");
  const currentValue =
    activeEnvFilter && typeof activeEnvFilter.value === "string" ? activeEnvFilter.value : "all";

  const handleChange = (value: string) => {
    const otherFilters = filters.filter((f) => f.field !== "environment");
    if (value === "all") {
      updateFilters(otherFilters);
    } else {
      updateFilters([
        ...otherFilters,
        {
          field: "environment",
          id: crypto.randomUUID(),
          operator: "is",
          value,
        },
      ]);
    }
  };

  return (
    <Select value={currentValue} onValueChange={handleChange}>
      <SelectTrigger
        className="h-9 w-full bg-gray-1"
        leftIcon={<Layers3 iconSize="md-medium" className="text-gray-9" />}
        rightIcon={<ChevronDown className="absolute right-2" iconSize="md-medium" />}
      >
        <SelectValue placeholder="All Environments" />
      </SelectTrigger>
      <SelectContent>
        <SelectItem value="all">All Environments</SelectItem>
        {environments.map((env) => (
          <SelectItem key={env.id} value={env.slug}>
            {env.slug.charAt(0).toUpperCase() + env.slug.slice(1)}
          </SelectItem>
        ))}
      </SelectContent>
    </Select>
  );
}
