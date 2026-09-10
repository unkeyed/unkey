"use client";

import {
  Multibox,
  MultiboxChip,
  MultiboxChipRemove,
  MultiboxChips,
  MultiboxContent,
  MultiboxEmpty,
  MultiboxInput,
  MultiboxItem,
  MultiboxList,
  MultiboxTrigger,
  useMultiboxAnchor,
} from "@/components/ui/multibox";
import { Plus, Trash } from "@unkey/icons";
import { unkeyAuditLogEvents } from "@unkey/schema/src/auditlog";
import { Button, FormInput } from "@unkey/ui";
import { Controller, useFieldArray, useFormContext, useWatch } from "react-hook-form";
import { DrainEndpointRow } from "./drain-endpoint-row";
import { type DrainFormValues, emptyHeaderRow } from "./drain-schema";

export function NameField() {
  const { register, formState } = useFormContext<DrainFormValues>();

  return (
    <FormInput
      requirement="required"
      label="Name"
      description="Shown in the log drain list and on the drain page."
      className="[&_input:first-of-type]:h-[36px]"
      placeholder="Production audit logs"
      error={formState.errors.name?.message}
      {...register("name")}
    />
  );
}

export function EventTypesField() {
  const { control } = useFormContext<DrainFormValues>();
  const anchor = useMultiboxAnchor();

  return (
    <Controller
      control={control}
      name="eventTypes"
      render={({ field }) => {
        const choices = Array.from(new Set([...unkeyAuditLogEvents.options, ...field.value]));
        return (
          <fieldset className="flex flex-col gap-1.5">
            <legend className="text-[13px] text-gray-11">Event types</legend>
            <span className="text-xs text-gray-9">
              Choose which audit events to send. Leave empty to send all events, including new event
              types added later.
            </span>
            <Multibox items={choices} value={field.value} onValueChange={field.onChange}>
              <MultiboxChips ref={anchor} className="mt-1.5">
                {field.value.map((eventType) => (
                  <MultiboxChip key={eventType}>
                    <span className="font-mono">{eventType}</span>
                    <MultiboxChipRemove />
                  </MultiboxChip>
                ))}
                <MultiboxInput
                  aria-label="Search event types"
                  placeholder={field.value.length === 0 ? "All event types" : "Search events"}
                  onBlur={field.onBlur}
                />
                <MultiboxTrigger />
              </MultiboxChips>
              <MultiboxContent anchor={anchor}>
                <MultiboxEmpty>No event types found.</MultiboxEmpty>
                <MultiboxList>
                  {(eventType: string) => (
                    <MultiboxItem key={eventType} value={eventType}>
                      <span className="font-mono text-xs">{eventType}</span>
                    </MultiboxItem>
                  )}
                </MultiboxList>
              </MultiboxContent>
            </Multibox>
          </fieldset>
        );
      }}
    />
  );
}

export function HeaderFields() {
  const { control, register, formState } = useFormContext<DrainFormValues>();
  const { fields, append, remove } = useFieldArray({ control, name: "headers" });
  const errors = formState.errors.headers;

  return (
    <fieldset className="flex flex-col gap-1.5">
      <legend className="text-[13px] text-gray-11">Headers</legend>
      <span className="text-xs text-gray-9">
        Optional. Unkey encrypts header values before storing them, and hides them afterwards.
      </span>
      <div className="mt-1.5 flex flex-col gap-3">
        {fields.map((field, index) => (
          <div key={field.id} className="flex items-start gap-3">
            <FormInput
              label="Name"
              placeholder="Authorization"
              className="flex-1 [&_input:first-of-type]:h-[36px]"
              // A stored header is addressed by name on save, so renaming it cannot mean anything.
              readOnly={field.stored}
              error={errors?.[index]?.name?.message}
              {...register(`headers.${index}.name`)}
            />
            <FormInput
              label="Value"
              type="password"
              autoComplete="off"
              placeholder={field.stored ? "•••••••••• unchanged" : "Bearer …"}
              className="flex-1 [&_input:first-of-type]:h-[36px]"
              error={errors?.[index]?.value?.message}
              {...register(`headers.${index}.value`)}
            />
            <Button
              type="button"
              variant="ghost"
              size="sm"
              className="mt-[26px] size-9 shrink-0 justify-center px-0 text-gray-11"
              aria-label={`Remove header ${index + 1}`}
              onClick={() => remove(index)}
            >
              <Trash iconSize="sm-regular" />
            </Button>
          </div>
        ))}
        <Button
          type="button"
          variant="outline"
          className="w-fit"
          disabled={fields.length >= 32}
          onClick={() => append({ ...emptyHeaderRow })}
        >
          <Plus iconSize="sm-regular" />
          Add header
        </Button>
      </div>
    </fieldset>
  );
}

function HttpFields() {
  return (
    <>
      <DrainEndpointRow />
      <HeaderFields />
    </>
  );
}

function AxiomFields({ tokenRequired }: { tokenRequired: boolean }) {
  const { register, formState } = useFormContext<DrainFormValues>();

  return (
    <>
      <FormInput
        requirement="required"
        label="Dataset"
        description="The Axiom dataset that receives audit logs."
        className="[&_input:first-of-type]:h-[36px]"
        placeholder="audit-logs"
        error={formState.errors.dataset?.message}
        {...register("dataset")}
      />
      <FormInput
        requirement={tokenRequired ? "required" : "optional"}
        label="Token"
        description={
          tokenRequired
            ? "Use an Axiom API token that can ingest data into this dataset."
            : "Leave blank to keep the current token."
        }
        type="password"
        autoComplete="off"
        placeholder={tokenRequired ? undefined : "•••••••••• unchanged"}
        className="[&_input:first-of-type]:h-[36px]"
        error={formState.errors.token?.message}
        {...register("token")}
      />
    </>
  );
}

export function DestinationFields({ tokenRequired }: { tokenRequired: boolean }) {
  const kind = useWatch<DrainFormValues, "kind">({ name: "kind" });

  return kind === "http" ? <HttpFields /> : <AxiomFields tokenRequired={tokenRequired} />;
}
