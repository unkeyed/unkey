"use client";

import type { ComboboxOption } from "@/components/ui/combobox";
import { FormCombobox } from "@/components/ui/form-combobox";
import { trpc } from "@/lib/trpc/client";
import { Plus, Trash, XMark } from "@unkey/icons";
import { Button, FormInput } from "@unkey/ui";
import { SimpleSelect } from "../simple-select";
import type { KeyAuthConfig } from "../types";

export function KeyAuthForm({
  config,
  onChange,
}: {
  config: KeyAuthConfig;
  onChange: (config: KeyAuthConfig) => void;
}) {
  const { data: availableKeyspaces } =
    trpc.deploy.environmentSettings.getAvailableKeyspaces.useQuery();

  const ksMap = availableKeyspaces ?? {};
  const unselectedKeyspaceIds = Object.keys(ksMap).filter((id) => !config.keySpaceIds.includes(id));

  const comboboxOptions: ComboboxOption[] = unselectedKeyspaceIds.map((id) => ({
    value: id,
    searchValue: `${id} ${ksMap[id]?.api?.name ?? ""}`,
    label: <span className="text-gray-11 text-xs font-mono">{ksMap[id]?.api?.name ?? id}</span>,
  }));

  const addKeyspace = (id: string) => {
    if (id && !config.keySpaceIds.includes(id)) {
      onChange({ ...config, keySpaceIds: [...config.keySpaceIds, id] });
    }
  };

  const removeKeyspace = (id: string) => {
    onChange({ ...config, keySpaceIds: config.keySpaceIds.filter((k) => k !== id) });
  };

  const addLocation = () => {
    onChange({ ...config, locations: [...config.locations, { type: "bearer" }] });
  };

  const removeLocation = (index: number) => {
    onChange({ ...config, locations: config.locations.filter((_, i) => i !== index) });
  };

  const updateLocation = (index: number, loc: KeyAuthConfig["locations"][number]) => {
    const locs = [...config.locations];
    locs[index] = loc;
    onChange({ ...config, locations: locs });
  };

  return (
    <div className="flex flex-col gap-4">
      <FormCombobox
        label="Keyspaces"
        options={comboboxOptions}
        value=""
        onSelect={addKeyspace}
        placeholder={
          config.keySpaceIds.length === 0 ? (
            <span className="text-grayA-8 w-full text-left">Select a keyspace</span>
          ) : (
            <div className="w-full flex flex-wrap gap-1.5 py-0.5">
              {config.keySpaceIds.map((id) => (
                <span
                  key={id}
                  className="flex items-center gap-1 px-1.5 py-0.5 rounded-md bg-grayA-3 border border-grayA-4 text-xs text-accent-12"
                >
                  {ksMap[id]?.api?.name ?? id}
                  <button
                    type="button"
                    onClick={(e) => {
                      e.stopPropagation();
                      removeKeyspace(id);
                    }}
                    className="p-0.5 hover:bg-grayA-4 rounded text-grayA-9 hover:text-accent-12 transition-colors"
                  >
                    <XMark iconSize="sm-regular" />
                  </button>
                </span>
              ))}
            </div>
          )
        }
        searchPlaceholder="Search keyspaces..."
        emptyMessage={<div className="mt-2">No keyspaces available.</div>}
      />

      <div>
        <div className="flex items-center justify-between mb-2">
          <span className="text-xs font-medium text-gray-11">Key Locations</span>
          <Button variant="ghost" size="sm" onClick={addLocation}>
            <Plus className="size-3" />
            Add
          </Button>
        </div>
        {config.locations.length === 0 && (
          <p className="text-xs text-grayA-8">Default: Bearer token from Authorization header</p>
        )}
        <div className="flex flex-col gap-2">
          {config.locations.map((loc, i) => (
            <div key={`${loc.type}-${i}`} className="flex items-center gap-2">
              <SimpleSelect
                value={loc.type}
                options={[
                  { value: "bearer", label: "Bearer" },
                  { value: "header", label: "Header" },
                  { value: "queryParam", label: "Query Param" },
                ]}
                onChange={(v) => {
                  const type = v as "bearer" | "header" | "queryParam";
                  if (type === "bearer") {
                    updateLocation(i, { type: "bearer" });
                  } else {
                    updateLocation(i, { type, name: "" } as KeyAuthConfig["locations"][number]);
                  }
                }}
              />
              {loc.type === "header" && (
                <div className="flex-1">
                  <FormInput
                    placeholder="X-API-Key"
                    value={loc.name}
                    onChange={(e) => updateLocation(i, { ...loc, name: e.target.value })}
                  />
                </div>
              )}
              {loc.type === "queryParam" && (
                <div className="flex-1">
                  <FormInput
                    placeholder="api_key"
                    value={loc.name}
                    onChange={(e) => updateLocation(i, { ...loc, name: e.target.value })}
                  />
                </div>
              )}
              <Button
                variant="ghost"
                size="sm"
                className="text-grayA-8 hover:text-red-10 shrink-0"
                onClick={() => removeLocation(i)}
              >
                <Trash className="size-3" />
              </Button>
            </div>
          ))}
        </div>
      </div>

      <FormInput
        label="Permission Query"
        value={config.permissionQuery}
        placeholder="api.read AND api.write"
        onChange={(e) => onChange({ ...config, permissionQuery: e.target.value })}
        description="RBAC permission expression (optional)"
      />
    </div>
  );
}
