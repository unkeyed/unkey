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
import { KEY_VERIFICATION_OUTCOMES } from "@unkey/clickhouse/src/keys/keys";
import { Plus, Trash } from "@unkey/icons";
import { match } from "@unkey/match";
import { unkeyAuditLogEvents } from "@unkey/schema/src/auditlog";
import { Button, FormInput, FormSelect } from "@unkey/ui";
import type { ReactNode } from "react";
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

export function StreamField({ disabled = false }: { disabled?: boolean }) {
  const { control } = useFormContext<DrainFormValues>();
  return (
    <Controller
      control={control}
      name="stream"
      render={({ field }) => (
        <FormSelect
          label="Stream"
          value={field.value}
          onValueChange={field.onChange}
          disabled={disabled}
          options={[
            { value: "audit_logs", label: "Audit logs" },
            { value: "key_verifications", label: "Key verifications" },
          ]}
        />
      )}
    />
  );
}

export function EventTypesField() {
  const { control } = useFormContext<DrainFormValues>();
  const stream = useWatch({ control, name: "stream" });

  return match(stream)
    .with("audit_logs", () => <AuditEventTypesField />)
    .with("key_verifications", () => <VerificationOutcomesField />)
    .exhaustive();
}

function AuditEventTypesField() {
  const { control } = useFormContext<DrainFormValues>();
  return (
    <Controller
      control={control}
      name="eventTypes"
      render={({ field }) => (
        <FilterChoices
          {...field}
          options={unkeyAuditLogEvents.options}
          label="Event types"
          description={
            <>
              Choose which audit events to send.{" "}
              <a
                href="https://www.unkey.com/docs/audit-log/types"
                target="_blank"
                rel="noopener noreferrer"
                className="underline underline-offset-2"
              >
                View event types
              </a>
            </>
          }
          searchLabel="Search event types"
          placeholder="All event types"
          emptyMessage="No event types found."
        />
      )}
    />
  );
}

function VerificationOutcomesField() {
  const { control } = useFormContext<DrainFormValues>();
  return (
    <Controller
      control={control}
      name="outcomes"
      render={({ field }) => (
        <FilterChoices
          {...field}
          options={KEY_VERIFICATION_OUTCOMES}
          label="Outcomes"
          description="Choose which verification outcomes to send. Leave empty to send all outcomes."
          searchLabel="Search outcomes"
          placeholder="All outcomes"
          emptyMessage="No outcomes found."
        />
      )}
    />
  );
}

function FilterChoices({
  value,
  onChange,
  onBlur,
  options,
  label,
  description,
  searchLabel,
  placeholder,
  emptyMessage,
}: {
  value: string[];
  onChange: (value: string[]) => void;
  onBlur: () => void;
  options: readonly string[];
  label: string;
  description: ReactNode;
  searchLabel: string;
  placeholder: string;
  emptyMessage: string;
}) {
  const anchor = useMultiboxAnchor();
  const choices = Array.from(new Set([...options, ...value]));

  return (
    <fieldset className="flex flex-col gap-1.5">
      <legend className="text-[13px] text-gray-11">{label}</legend>
      <span className="text-xs text-gray-9">{description}</span>
      <Multibox items={choices} value={value} onValueChange={onChange}>
        <MultiboxChips ref={anchor} className="mt-1.5">
          {value.map((choice) => (
            <MultiboxChip key={choice}>
              <span className="font-mono">{choice || "Unspecified"}</span>
              <MultiboxChipRemove />
            </MultiboxChip>
          ))}
          <MultiboxInput
            aria-label={searchLabel}
            placeholder={value.length === 0 ? placeholder : "Search"}
            onBlur={onBlur}
          />
          <MultiboxTrigger />
        </MultiboxChips>
        <MultiboxContent anchor={anchor}>
          <MultiboxEmpty>{emptyMessage}</MultiboxEmpty>
          <MultiboxList>
            {(choice: string) => (
              <MultiboxItem key={choice} value={choice}>
                <span className="font-mono text-xs">{choice || "Unspecified"}</span>
              </MultiboxItem>
            )}
          </MultiboxList>
        </MultiboxContent>
      </Multibox>
    </fieldset>
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
        description="The Axiom dataset that receives this stream."
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
