"use client";

import type { ComboboxOption } from "@/components/ui/combobox";
import { FormCombobox } from "@/components/ui/form-combobox";
import { trpc } from "@/lib/trpc/client";
import { ChevronDown, ChevronUp, Plus, Trash, XMark } from "@unkey/icons";
import { match } from "@unkey/match";
import {
  Button,
  FormInput,
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@unkey/ui";
import { FormDescription } from "@unkey/ui/src/components/form/form-helpers";
import type { Control } from "react-hook-form";
import { useController } from "react-hook-form";
import type { KeyLocationFormValues, KeyLocationType, PolicyFormValues } from "../schema";

type KeyauthFormValues = Extract<PolicyFormValues, { type: "keyauth" }>;

const LOCATION_TYPE_OPTIONS: { value: KeyLocationType; label: string }[] = [
  { value: "bearer", label: "Bearer" },
  { value: "header", label: "Header" },
  { value: "queryParam", label: "Query Param" },
];

export function KeyAuthFields({ control }: { control: Control<KeyauthFormValues> }) {
  const {
    field: { value: keySpaceIds, onChange: setKeySpaceIds },
    fieldState: { error: keySpaceError },
  } = useController({ control, name: "keySpaceIds" });

  const {
    field: { value: locations, onChange: setLocations },
  } = useController({ control, name: "locations" });

  const {
    field: { value: permissionQuery, onChange: setPermissionQuery },
  } = useController({ control, name: "permissionQuery" });

  const { data: availableKeyspaces = {} } =
    trpc.deploy.environmentSettings.getAvailableKeyspaces.useQuery();

  const unselected = Object.keys(availableKeyspaces).filter((id) => !keySpaceIds.includes(id));
  const comboboxOptions: ComboboxOption[] = unselected.map((id) => ({
    value: id,
    searchValue: id,
    label: (
      <span className="text-gray-11 text-xs font-mono">
        {availableKeyspaces[id]?.api?.name ?? id}
      </span>
    ),
  }));

  const addLocation = () => {
    setLocations([...locations, { id: crypto.randomUUID(), locationType: "bearer" }]);
  };

  const updateLocation = (id: string, updates: Partial<KeyLocationFormValues>) => {
    setLocations(locations.map((loc) => (loc.id === id ? { ...loc, ...updates } : loc)));
  };

  const removeLocation = (id: string) => {
    setLocations(locations.filter((loc) => loc.id !== id));
  };

  const moveLocation = (index: number, direction: -1 | 1) => {
    const target = index + direction;
    if (target < 0 || target >= locations.length) {
      return;
    }
    const next = [...locations];
    [next[index], next[target]] = [next[target], next[index]];
    setLocations(next);
  };

  return (
    <div className="flex flex-col gap-5">
      <div className="flex flex-col gap-1.5">
        <FormCombobox
          label="Keyspaces"
          description="API keyspaces used to authenticate incoming requests."
          options={comboboxOptions}
          value=""
          onSelect={(id) => {
            if (!keySpaceIds.includes(id)) {
              setKeySpaceIds([...keySpaceIds, id]);
            }
          }}
          placeholder={
            keySpaceIds.length === 0 ? (
              <span className="text-grayA-8 w-full text-left">Select a keyspace</span>
            ) : (
              <div className="w-full flex flex-wrap gap-1.5 py-0.5">
                {keySpaceIds.map((id) => (
                  <span
                    key={id}
                    className="flex items-center gap-1 px-1.5 py-0.5 rounded-md bg-grayA-3 border border-grayA-4 text-xs text-accent-12"
                  >
                    {availableKeyspaces[id]?.api?.name ?? id}
                    <button
                      type="button"
                      onClick={(e) => {
                        e.stopPropagation();
                        setKeySpaceIds(keySpaceIds.filter((k) => k !== id));
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
        {keySpaceError && <p className="text-error-11 text-[13px]">{keySpaceError.message}</p>}
      </div>

      <fieldset className="flex flex-col gap-2 border-0 m-0 p-0">
        <div className="flex items-center justify-between">
          {/* biome-ignore lint/a11y/noLabelWithoutControl: its okay */}
          <label className="text-gray-11 text-[13px]">Key Locations</label>
          <Button type="button" variant="ghost" size="sm" onClick={addLocation}>
            <Plus iconSize="sm-regular" />
            Add
          </Button>
        </div>
        <FormDescription
          description={
            locations.length === 0
              ? "Where to extract the API key from. Locations are tried in order and the first non-empty value wins. Defaults to Bearer token if none configured."
              : "Tried in order. The first location that yields a non-empty key is used."
          }
          descriptionId="key-locations-desc"
          errorId="key-locations-error"
        />
        {locations.length > 0 && (
          <div className="flex flex-col gap-2">
            {locations.map((loc, index) => (
              <div key={loc.id} className="flex items-center gap-2">
                <div className="flex flex-col shrink-0">
                  <button
                    type="button"
                    aria-label="Move up"
                    disabled={index === 0}
                    onClick={() => moveLocation(index, -1)}
                    className="text-gray-9 hover:text-gray-12 disabled:opacity-30 disabled:cursor-default transition-colors p-0.5"
                  >
                    <ChevronUp iconSize="sm-regular" />
                  </button>
                  <button
                    type="button"
                    aria-label="Move down"
                    disabled={index === locations.length - 1}
                    onClick={() => moveLocation(index, 1)}
                    className="text-gray-9 hover:text-gray-12 disabled:opacity-30 disabled:cursor-default transition-colors p-0.5"
                  >
                    <ChevronDown iconSize="sm-regular" />
                  </button>
                </div>
                <span className="text-[11px] text-gray-9 w-4 shrink-0 text-right tabular-nums">
                  {index + 1}.
                </span>
                <div className="w-32 shrink-0">
                  <Select
                    value={loc.locationType}
                    onValueChange={(v) => {
                      const locationType = v as KeyLocationType;
                      updateLocation(loc.id, {
                        locationType,
                        name: locationType === "bearer" ? undefined : "",
                        stripPrefix: undefined,
                      });
                    }}
                  >
                    <SelectTrigger
                      aria-label="Location type"
                      className="shrink-0 whitespace-pre"
                      rightIcon={<ChevronDown className="absolute right-2" iconSize="md-medium" />}
                    >
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent className="z-60">
                      {LOCATION_TYPE_OPTIONS.map((opt) => (
                        <SelectItem
                          key={opt.value}
                          value={opt.value}
                          className="shrink-0 whitespace-pre"
                        >
                          {opt.label}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                </div>
                {match(loc.locationType)
                  .with("bearer", () => (
                    <span className="flex-1 text-[12px] text-gray-9">
                      Authorization: Bearer &lt;key&gt;
                    </span>
                  ))
                  .with("header", () => (
                    <FormInput
                      placeholder="X-API-Key"
                      value={loc.name ?? ""}
                      onChange={(e) => updateLocation(loc.id, { name: e.target.value })}
                      className="flex-1"
                    />
                  ))
                  .with("queryParam", () => (
                    <FormInput
                      placeholder="api_key"
                      value={loc.name ?? ""}
                      onChange={(e) => updateLocation(loc.id, { name: e.target.value })}
                      className="flex-1"
                    />
                  ))
                  .exhaustive()}
                <Button
                  type="button"
                  variant="ghost"
                  size="sm"
                  aria-label="Remove location"
                  className="size-9 shrink-0 px-0 justify-center text-gray-11 hover:text-gray-12 hover:bg-grayA-3 rounded-lg"
                  onClick={() => removeLocation(loc.id)}
                >
                  <Trash iconSize="sm-regular" />
                </Button>
              </div>
            ))}
          </div>
        )}
      </fieldset>

      <FormInput
        label="Permission Query"
        placeholder="e.g. api.read AND api.write"
        value={permissionQuery}
        onChange={(e) => setPermissionQuery(e.target.value)}
        description={
          <span>
            Reject requests if the key lacks permissions. Supports{" "}
            <span className="text-gray-12 font-medium">AND</span> /{" "}
            <span className="text-gray-12 font-medium">OR</span> operators.
          </span>
        }
      />
    </div>
  );
}
