"use client";

import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@unkey/ui";
import { IconChevronDownOutline18, IconLayers3Outline18 } from "nucleo-ui-outline-18";
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
    <Select
      value={currentValue}
      items={[
        { value: "all", label: "All Environments" },
        ...environments.map((env) => ({
          value: env.slug,
          label: env.slug.charAt(0).toUpperCase() + env.slug.slice(1),
        })),
      ]}
      onValueChange={(newValue) => {
        if (newValue !== null) {
          handleChange(newValue);
        }
      }}
    >
      <SelectTrigger
        className="h-9 w-full bg-gray-1"
        leftIcon={<IconLayers3Outline18 className="size-4 text-gray-9" />}
        rightIcon={<IconChevronDownOutline18 className="size-4 absolute right-2" />}
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
