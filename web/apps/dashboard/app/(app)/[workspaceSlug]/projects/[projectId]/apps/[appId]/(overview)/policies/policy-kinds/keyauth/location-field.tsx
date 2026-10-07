"use client";

import { IconChevronDownOutline18 } from "@unkey/icons";
import { match } from "@unkey/match";
import {
  FormDescription,
  FormInput,
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@unkey/ui";
import { useController, useFormContext, useFormState } from "react-hook-form";
import { FieldSection, RemoveRowButton } from "../field-parts";
import type { KeyLocationFormValues, KeyLocationType, KeyauthFormValues } from "./model";

const LOCATION_TYPE_OPTIONS: { value: KeyLocationType; label: string }[] = [
  { value: "bearer", label: "Bearer" },
  { value: "header", label: "Header" },
  { value: "queryParam", label: "Query Param" },
];

export function KeyLocationField() {
  const { control, setValue } = useFormContext<KeyauthFormValues>();
  const { errors, isSubmitted } = useFormState({ control });
  const {
    field: { value: locations, onChange: setLocations },
  } = useController({ control, name: "locations" });

  const location = locations[0];
  const nameError = errors.locations?.[0]?.name;

  const update = (id: string, updates: Partial<KeyLocationFormValues>) => {
    const next = locations.map((loc) => (loc.id === id ? { ...loc, ...updates } : loc));
    setValue("locations", next, { shouldDirty: true, shouldValidate: isSubmitted });
  };

  return (
    <FieldSection
      label="Key Location"
      htmlFor="key-locations"
      tooltip="Defaults to Bearer token."
      onAdd={
        location ? null : () => setLocations([{ id: crypto.randomUUID(), locationType: "bearer" }])
      }
    >
      {location ? (
        <>
          <div className="flex items-center gap-2">
            <div className="w-32 shrink-0">
              <Select
                value={location.locationType}
                items={LOCATION_TYPE_OPTIONS}
                onValueChange={(v) => {
                  const locationType = LOCATION_TYPE_OPTIONS.find((o) => o.value === v)?.value;
                  if (!locationType) {
                    return;
                  }
                  update(location.id, {
                    locationType,
                    name: locationType === "bearer" ? undefined : "",
                    stripPrefix: undefined,
                  });
                }}
              >
                <SelectTrigger
                  aria-label="Location type"
                  className="shrink-0 whitespace-pre"
                  rightIcon={<IconChevronDownOutline18 className="size-3.5 absolute right-2" />}
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
                <span className="flex-1 text-xs text-gray-9">
                  Authorization: Bearer &lt;key&gt;
                </span>
              ))
              .with("header", "queryParam", (type) => (
                <FormInput
                  placeholder={type === "header" ? "X-API-Key" : "api_key"}
                  value={location.name ?? ""}
                  onChange={(e) => update(location.id, { name: e.target.value })}
                  className="flex-1"
                  variant={nameError ? "error" : undefined}
                  aria-invalid={Boolean(nameError)}
                />
              ))
              .exhaustive()}
            <RemoveRowButton label="Remove location" onClick={() => setLocations([])} />
          </div>
          <FormDescription
            error={nameError?.message}
            descriptionId="location-name-desc"
            errorId="location-name-error"
          />
        </>
      ) : null}
    </FieldSection>
  );
}
