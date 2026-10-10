"use client";

import { IconLayers3Outline18 } from "@unkey/icons";
import {
  ALL_ENVIRONMENTS,
  EnvironmentSelect,
} from "../../../../../../_components/environment-select";
import { useProjectData } from "../../../../data-provider";
import { useFilters } from "../../../hooks/use-filters";

export function EnvironmentFilter() {
  const { environments } = useProjectData();
  const { filters, updateFilters } = useFilters();

  const activeSlug = filters.find((f) => f.field === "environment")?.value;
  const activeId = environments.find((env) => env.slug === activeSlug)?.id ?? ALL_ENVIRONMENTS;

  const handleChange = (id: string) => {
    const otherFilters = filters.filter((f) => f.field !== "environment");
    const slug = environments.find((env) => env.id === id)?.slug;
    if (!slug) {
      updateFilters(otherFilters);
      return;
    }
    updateFilters([
      ...otherFilters,
      { field: "environment", id: crypto.randomUUID(), operator: "is", value: slug },
    ]);
  };

  return (
    <EnvironmentSelect
      value={activeId}
      onValueChange={handleChange}
      className="h-9 w-full bg-gray-1"
      leftIcon={<IconLayers3Outline18 className="size-3.5 text-gray-9" />}
    />
  );
}
