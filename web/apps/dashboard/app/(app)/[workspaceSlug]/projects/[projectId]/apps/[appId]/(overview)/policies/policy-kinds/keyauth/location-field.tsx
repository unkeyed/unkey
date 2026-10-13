"use client";

import { FormDescription, FormInput, FormSelect } from "@unkey/ui";
import { useFieldArray, useFormContext, useFormState, useWatch } from "react-hook-form";
import { FieldSection, RemoveRowButton } from "../field-parts";
import {
  KEY_LOCATION_TYPES,
  type KeyLocationFormValues,
  type KeyLocationOf,
  type KeyLocationType,
  type KeyauthFormValues,
  keyLocationDefaults,
} from "./model";

const KEY_LOCATIONS: {
  [T in KeyLocationType]: {
    label: string;
    namePlaceholder: KeyLocationOf<T> extends { name: string } ? string : null;
  };
} = {
  bearer: { label: "Bearer", namePlaceholder: null },
  header: { label: "Header", namePlaceholder: "X-API-Key" },
  queryParam: { label: "Query Param", namePlaceholder: "api_key" },
};

const LOCATION_TYPE_OPTIONS = KEY_LOCATION_TYPES.map((value) => ({
  value,
  label: KEY_LOCATIONS[value].label,
}));

export function KeyLocationField() {
  const { control, setValue, getFieldState } = useFormContext<KeyauthFormValues>();
  const { fields, append, remove } = useFieldArray({ control, name: "locations" });
  const locations = useWatch({ control, name: "locations" }) ?? [];
  const formState = useFormState({ control, name: "locations" });
  const update = (index: number, next: KeyLocationFormValues) =>
    setValue(`locations.${index}`, next, {
      shouldDirty: true,
      shouldValidate: formState.isSubmitted,
    });

  return (
    <FieldSection
      label="Key Location"
      htmlFor="key-locations"
      tooltip="Defaults to Bearer token. Several locations are tried in order."
      onAdd={() => append(keyLocationDefaults.bearer)}
    >
      {fields.map((field, index) => {
        const location = locations[index] ?? field;
        const nameError = getFieldState(`locations.${index}.name`, formState).error?.message;
        return (
          <div key={field.id} className="flex flex-col gap-1">
            <div className="flex items-center gap-2">
              <FormSelect
                aria-label="Location type"
                options={LOCATION_TYPE_OPTIONS}
                value={location.locationType}
                onValueChange={(type) => update(index, keyLocationDefaults[type])}
                className="w-32 shrink-0"
                triggerClassName="shrink-0 whitespace-pre"
                itemClassName="shrink-0 whitespace-pre"
              />
              {"name" in location ? (
                <FormInput
                  placeholder={KEY_LOCATIONS[location.locationType].namePlaceholder}
                  value={location.name}
                  onChange={(e) => update(index, { ...location, name: e.target.value })}
                  className="flex-1"
                  variant={nameError ? "error" : undefined}
                  aria-invalid={Boolean(nameError)}
                />
              ) : (
                <span className="flex-1 text-xs text-gray-9">
                  Authorization: Bearer &lt;key&gt;
                </span>
              )}
              <RemoveRowButton label="Remove location" onClick={() => remove(index)} />
            </div>
            {nameError ? (
              <FormDescription
                error={nameError}
                descriptionId={`location-${index}-desc`}
                errorId={`location-${index}-error`}
              />
            ) : null}
          </div>
        );
      })}
    </FieldSection>
  );
}
