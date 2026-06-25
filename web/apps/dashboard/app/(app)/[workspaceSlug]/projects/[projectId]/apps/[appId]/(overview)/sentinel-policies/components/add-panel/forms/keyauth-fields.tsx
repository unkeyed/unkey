"use client";

import type { ComboboxOption } from "@/components/ui/combobox";
import { FormCombobox } from "@/components/ui/form-combobox";
import { Switch } from "@/components/ui/switch";
import { SENTINEL_LIMITS } from "@/lib/collections/deploy/sentinel-policies.schema";
import { trpc } from "@/lib/trpc/client";
import { ChevronDown, Plus, Trash, XMark } from "@unkey/icons";
import { match } from "@unkey/match";
import {
  Button,
  FormDescription,
  FormInput,
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@unkey/ui";
import { FormLabel } from "@unkey/ui/src/components/form/form-helpers";
import type { ReactNode } from "react";
import { useController, useFormContext, useFormState, useWatch } from "react-hook-form";
import type {
  KeyLocationFormValues,
  KeyLocationType,
  KeyauthRatelimitFormValues,
  PolicyFormValues,
} from "../schema";
import { Sep, Strong } from "./summary-helpers";

type KeyauthFormValues = Extract<PolicyFormValues, { type: "keyauth" }>;

const LOCATION_TYPE_OPTIONS: { value: KeyLocationType; label: string }[] = [
  { value: "bearer", label: "Bearer" },
  { value: "header", label: "Header" },
  { value: "queryParam", label: "Query Param" },
];

export function KeyAuthFields() {
  const { control, setValue } = useFormContext<KeyauthFormValues>();
  const { errors, isSubmitted } = useFormState({ control });

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

  const {
    field: { value: ratelimits, onChange: setRatelimits },
  } = useController({ control, name: "ratelimits" });

  const locationErrors = errors.locations as
    | Record<number, Partial<Record<string, { message?: string }>>>
    | undefined;
  const nameError = locationErrors?.[0]?.name;

  const ratelimitErrors = errors.ratelimits as
    | Record<number, Partial<Record<string, { message?: string }>>>
    | undefined;

  const addRatelimit = () => {
    setRatelimits([...ratelimits, { id: crypto.randomUUID(), name: "", override: false }]);
  };

  const updateRatelimit = (id: string, updates: Partial<KeyauthRatelimitFormValues>) => {
    setValue(
      "ratelimits",
      ratelimits.map((rl) => (rl.id === id ? { ...rl, ...updates } : rl)),
      { shouldDirty: true, shouldValidate: isSubmitted },
    );
  };

  // Toggling the override off clears the override values so a disabled override
  // never leaks onto the wire.
  const setRatelimitOverride = (id: string, enabled: boolean) => {
    updateRatelimit(
      id,
      enabled
        ? { override: true }
        : { override: false, limit: undefined, duration: undefined, cost: undefined },
    );
  };

  const removeRatelimit = (id: string) => {
    setRatelimits(ratelimits.filter((rl) => rl.id !== id));
  };

  // Number inputs map empty → undefined so a blank field references the named
  // limit on the key rather than coercing to 0.
  const toOptionalInt = (raw: string): number | undefined => {
    if (raw === "") {
      return undefined;
    }
    const n = Number.parseInt(raw, 10);
    return Number.isNaN(n) ? undefined : n;
  };

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
    setLocations([{ id: crypto.randomUUID(), locationType: "bearer" }]);
  };

  const updateLocation = (id: string, updates: Partial<KeyLocationFormValues>) => {
    const next = locations.map((loc) => (loc.id === id ? { ...loc, ...updates } : loc));
    setValue("locations", next, { shouldDirty: true, shouldValidate: isSubmitted });
  };

  const removeLocation = () => {
    setLocations([]);
  };

  const location = locations[0];

  return (
    <div className="flex flex-col gap-5">
      <div className="flex flex-col gap-1.5">
        <FormCombobox
          label="Keyspaces"
          descriptionPosition="label"
          description="API keyspaces used to authenticate incoming requests."
          error={keySpaceError?.message}
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
                    {/* biome-ignore lint/a11y/useSemanticElements: nested inside a <button> (combobox trigger), so <button> is invalid here */}
                    <span
                      role="button"
                      tabIndex={0}
                      aria-label={`Remove ${availableKeyspaces[id]?.api?.name ?? id}`}
                      onClick={(e) => {
                        e.stopPropagation();
                        setKeySpaceIds(keySpaceIds.filter((k) => k !== id));
                      }}
                      onKeyDown={(e) => {
                        if (e.key === "Enter" || e.key === " ") {
                          e.stopPropagation();
                          setKeySpaceIds(keySpaceIds.filter((k) => k !== id));
                        }
                      }}
                      className="p-0.5 hover:bg-grayA-4 rounded text-grayA-9 hover:text-accent-12 transition-colors cursor-pointer"
                    >
                      <XMark iconSize="sm-regular" />
                    </span>
                  </span>
                ))}
              </div>
            )
          }
          searchPlaceholder="Search keyspaces..."
          emptyMessage={<div className="mt-2">No keyspaces available.</div>}
        />
      </div>

      <fieldset className="flex flex-col gap-2 border-0 m-0 p-0">
        <div className="flex items-center justify-between">
          <FormLabel
            label="Key Location"
            htmlFor="key-locations"
            tooltipContent="Where to extract the API key from. Defaults to Bearer token if not configured."
          />
          {!location && (
            <Button
              type="button"
              variant="outline"
              size="md"
              className="font-medium"
              onClick={addLocation}
            >
              <Plus iconSize="sm-regular" />
              Add
            </Button>
          )}
        </div>
        {location && (
          <>
            <div className="flex items-center gap-2">
              <div className="w-32 shrink-0">
                <Select
                  value={location.locationType}
                  onValueChange={(v) => {
                    const locationType = v as KeyLocationType;
                    updateLocation(location.id, {
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
                  <SelectContent>
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
              {match(location.locationType)
                .with("bearer", () => (
                  <span className="flex-1 text-[12px] text-gray-9">
                    Authorization: Bearer &lt;key&gt;
                  </span>
                ))
                .with("header", "queryParam", () => (
                  <FormInput
                    placeholder={location.locationType === "header" ? "X-API-Key" : "api_key"}
                    requirement="required"
                    value={location.name ?? ""}
                    onChange={(e) => updateLocation(location.id, { name: e.target.value })}
                    className="flex-1"
                    variant={nameError ? "error" : undefined}
                    aria-invalid={Boolean(nameError)}
                  />
                ))
                .exhaustive()}
              <Button
                type="button"
                variant="ghost"
                size="sm"
                aria-label="Remove location"
                className="size-9 shrink-0 px-0 justify-center text-gray-11 hover:text-gray-12 hover:bg-grayA-3 rounded-lg"
                onClick={removeLocation}
              >
                <Trash iconSize="sm-regular" />
              </Button>
            </div>
            <FormDescription
              error={nameError?.message}
              descriptionId="location-name-desc"
              errorId="location-name-error"
            />
          </>
        )}
      </fieldset>

      <FormInput
        label="Permission Query"
        requirement="optional"
        placeholder="e.g. api.read AND api.write"
        value={permissionQuery}
        onChange={(e) => setPermissionQuery(e.target.value)}
        descriptionPosition="inline"
        description={
          <span>
            Reject requests if the key lacks permissions. Supports{" "}
            <span className="text-gray-12 font-medium">AND</span> /{" "}
            <span className="text-gray-12 font-medium">OR</span> operators.
          </span>
        }
      />

      <fieldset className="flex flex-col gap-2 border-0 m-0 p-0">
        <div className="flex items-center justify-between">
          <FormLabel
            label="Key Rate Limits"
            htmlFor="keyauth-ratelimits"
            tooltipContent="Rate limits configured on the verified key to enforce for matching requests, mirroring verifyKey's ratelimits. Reference a limit by name; optionally override its limit, duration, and cost."
          />
          {ratelimits.length < SENTINEL_LIMITS.maxRatelimitsPerKeyauth && (
            <Button
              type="button"
              variant="outline"
              size="md"
              className="font-medium"
              onClick={addRatelimit}
            >
              <Plus iconSize="sm-regular" />
              Add
            </Button>
          )}
        </div>
        {ratelimits.map((rl, idx) => {
          const rowErr = ratelimitErrors?.[idx];
          const rowMessage =
            rowErr?.name?.message ??
            rowErr?.limit?.message ??
            rowErr?.duration?.message ??
            rowErr?.cost?.message;
          return (
            <div key={rl.id} className="flex flex-col gap-1.5">
              <div className="flex items-center gap-2">
                <FormInput
                  placeholder="name (e.g. expensive)"
                  requirement="required"
                  value={rl.name}
                  onChange={(e) => updateRatelimit(rl.id, { name: e.target.value })}
                  className="flex-1"
                  variant={rowErr?.name ? "error" : undefined}
                  aria-invalid={Boolean(rowErr?.name)}
                />
                <div className="flex items-center gap-1.5 shrink-0 text-[12px] text-gray-9">
                  <Switch
                    size="sm"
                    checked={rl.override}
                    onCheckedChange={(checked) => setRatelimitOverride(rl.id, checked)}
                    aria-label="Override limit"
                  />
                  Override
                </div>
                <Button
                  type="button"
                  variant="ghost"
                  size="sm"
                  aria-label="Remove rate limit"
                  className="size-9 shrink-0 px-0 justify-center text-gray-11 hover:text-gray-12 hover:bg-grayA-3 rounded-lg"
                  onClick={() => removeRatelimit(rl.id)}
                >
                  <Trash iconSize="sm-regular" />
                </Button>
              </div>
              {rl.override && (
                <div className="flex items-center gap-2 pl-1">
                  <FormInput
                    type="number"
                    placeholder="limit"
                    requirement="required"
                    value={rl.limit ?? ""}
                    onChange={(e) =>
                      updateRatelimit(rl.id, { limit: toOptionalInt(e.target.value) })
                    }
                    className="flex-1"
                    variant={rowErr?.limit ? "error" : undefined}
                    aria-invalid={Boolean(rowErr?.limit)}
                  />
                  <FormInput
                    type="number"
                    placeholder="duration (ms)"
                    requirement="required"
                    value={rl.duration ?? ""}
                    onChange={(e) =>
                      updateRatelimit(rl.id, { duration: toOptionalInt(e.target.value) })
                    }
                    className="flex-1"
                    variant={rowErr?.duration ? "error" : undefined}
                    aria-invalid={Boolean(rowErr?.duration)}
                  />
                  <FormInput
                    type="number"
                    placeholder="cost"
                    value={rl.cost ?? ""}
                    onChange={(e) =>
                      updateRatelimit(rl.id, { cost: toOptionalInt(e.target.value) })
                    }
                    className="w-20 shrink-0"
                    variant={rowErr?.cost ? "error" : undefined}
                    aria-invalid={Boolean(rowErr?.cost)}
                  />
                </div>
              )}
              {rowMessage && (
                <FormDescription
                  error={rowMessage}
                  descriptionId={`ratelimit-${rl.id}-desc`}
                  errorId={`ratelimit-${rl.id}-error`}
                />
              )}
            </div>
          );
        })}
        {ratelimits.length > 0 && (
          <FormDescription
            descriptionId="keyauth-ratelimits-desc"
            errorId="keyauth-ratelimits-error"
            description="Each entry enforces a rate limit defined on the key, referenced by name. Enable Override only to replace its limit, duration, and cost for this route."
          />
        )}
      </fieldset>
    </div>
  );
}

/**
 * Watches only the fields rendered in the summary so edits to unrelated
 * fields (name, environment) don't cause re-renders here.
 */
export function KeyauthPolicySummary() {
  const { control } = useFormContext<KeyauthFormValues>();
  const keySpaceIds = useWatch({ control, name: "keySpaceIds" });
  const locations = useWatch({ control, name: "locations" });
  const ratelimits = useWatch({ control, name: "ratelimits" });

  const { data: availableKeyspaces = {} } =
    trpc.deploy.environmentSettings.getAvailableKeyspaces.useQuery();
  const keyspaceNames: Record<string, string> = Object.fromEntries(
    Object.entries(availableKeyspaces).map(([id, ks]) => [id, ks?.api?.name ?? id]),
  );

  return (
    <div className="max-w-75 truncate">
      {summarizeKeyauth(keySpaceIds, locations, ratelimits, keyspaceNames)}
    </div>
  );
}

function summarizeKeyauth(
  keySpaceIds: string[],
  locations: KeyauthFormValues["locations"],
  ratelimits: KeyauthFormValues["ratelimits"],
  keyspaceNames?: Record<string, string>,
): ReactNode {
  return (
    <span className="text-gray-11">
      {keySpaceIds.length === 0 ? (
        <span className="text-gray-9">No keyspace selected</span>
      ) : keySpaceIds.length > 3 ? (
        <>
          <Strong>{keySpaceIds.length}</Strong> keyspaces
        </>
      ) : (
        <Strong className="inline-block max-w-50 truncate align-bottom">
          {keySpaceIds.map((id) => keyspaceNames?.[id] ?? id).join(", ")}
        </Strong>
      )}
      {locations.length === 1 && (
        <>
          <Sep />
          {summarizeLocation(locations[0])}
        </>
      )}
      {locations.length > 1 && (
        <>
          <Sep />
          <Strong>{locations.length}</Strong> key locations
        </>
      )}
      {ratelimits.length > 0 && (
        <>
          <Sep />
          <Strong>{ratelimits.length}</Strong>{" "}
          {ratelimits.length === 1 ? "rate limit" : "rate limits"}
        </>
      )}
    </span>
  );
}

function summarizeLocation(loc: KeyauthFormValues["locations"][number]): ReactNode {
  return match(loc.locationType)
    .with("bearer", () => <Strong>Bearer</Strong>)
    .with("header", () => (
      <>
        Header: <Strong>{loc.name || "—"}</Strong>
      </>
    ))
    .with("queryParam", () => (
      <>
        Query: <Strong>{loc.name || "—"}</Strong>
      </>
    ))
    .exhaustive();
}
