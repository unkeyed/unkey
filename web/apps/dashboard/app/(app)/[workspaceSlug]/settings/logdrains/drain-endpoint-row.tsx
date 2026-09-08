"use client";

import {
  FormField,
  FormSelect,
  InputGroup,
  InputGroupAddon,
  InputGroupInput,
  InputGroupText,
} from "@unkey/ui";
import { Controller, useFormContext } from "react-hook-form";
import type { DrainFormValues } from "./drain-schema";

const ENCODING_OPTIONS: Array<{
  value: DrainFormValues["format"];
  label: string;
}> = [
  { value: "json", label: "JSON" },
  { value: "ndjson", label: "NDJSON" },
];

export function DrainEndpointRow() {
  const { register, formState, control } = useFormContext<DrainFormValues>();

  return (
    <div className="flex items-start gap-3">
      <FormField label="URL" error={formState.errors.url?.message} className="min-w-0 flex-1">
        {(field) => (
          <InputGroup variant={field.variant}>
            <InputGroupAddon className="border-r border-gray-5 py-2 pr-3">
              <InputGroupText>POST</InputGroupText>
            </InputGroupAddon>
            <InputGroupInput
              id={field.id}
              placeholder="https://example.com/ingest"
              aria-invalid={field.invalid}
              aria-describedby={field.describedBy}
              {...register("url")}
            />
          </InputGroup>
        )}
      </FormField>
      <Controller
        control={control}
        name="format"
        render={({ field }) => (
          <FormSelect
            label="Encoding"
            className="shrink-0"
            options={ENCODING_OPTIONS}
            value={field.value}
            onValueChange={field.onChange}
            triggerClassName="h-9 w-32"
          />
        )}
      />
    </div>
  );
}
