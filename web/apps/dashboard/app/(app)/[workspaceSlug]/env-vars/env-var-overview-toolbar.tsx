"use client";

import { ResourceSearchInput } from "@/components/resource-search-input";
import { ToolbarSelect } from "@/components/toolbar-select";
import { ResourceListHeader } from "@unkey/ui";
import { TOOLBAR_FIELDS, type ToolbarState } from "./model";
import { OVERVIEW_SEARCH_KEY } from "./use-env-var-overview";

type ToolbarChange = (patch: Partial<ToolbarState>) => void;

export function EnvVarOverviewToolbar({
  toolbar,
  onChange,
}: {
  toolbar: ToolbarState;
  onChange: ToolbarChange;
}) {
  return (
    <ResourceListHeader>
      <ResourceSearchInput
        queryKey={OVERVIEW_SEARCH_KEY}
        label="Search environment variables"
        placeholder="Search by key"
      />
      <ToolbarField name="type" toolbar={toolbar} onChange={onChange} />
      <ToolbarField name="environment" toolbar={toolbar} onChange={onChange} />
      <ToolbarField name="sort" toolbar={toolbar} onChange={onChange} />
    </ResourceListHeader>
  );
}

function ToolbarField<K extends keyof ToolbarState>({
  name,
  toolbar,
  onChange,
}: {
  name: K;
  toolbar: ToolbarState;
  onChange: ToolbarChange;
}) {
  const field = TOOLBAR_FIELDS[name];
  return (
    <ToolbarSelect
      label={field.label}
      icon={<field.icon className="size-3.5 text-gray-9" />}
      value={toolbar[name]}
      options={field.options}
      onChange={(value) => {
        const patch: Partial<ToolbarState> = {};
        patch[name] = value;
        onChange(patch);
      }}
    />
  );
}
