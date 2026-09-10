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
import { trpc } from "@/lib/trpc/client";
import { Radio } from "@base-ui/react/radio";
import { RadioGroup } from "@base-ui/react/radio-group";
import { KEY_VERIFICATION_OUTCOMES } from "@unkey/clickhouse/src/keys/keys";
import { Plus, Trash } from "@unkey/icons";
import { match } from "@unkey/match";
import { unkeyAuditLogEvents } from "@unkey/schema/src/auditlog";
import { Button, FormInput, FormSelect, cn } from "@unkey/ui";
import { type ReactNode, useId } from "react";
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
    .with("key_verifications", () => (
      <>
        <VerificationKeyspacesField />
        <VerificationOutcomesField />
      </>
    ))
    .exhaustive();
}

const eventTypeModes = [
  { id: "all", title: "All event types" },
  { id: "specific", title: "Specific event types" },
] satisfies { id: DrainFormValues["eventTypesMode"]; title: string }[];

function AuditEventTypesField() {
  const { control, formState, setValue } = useFormContext<DrainFormValues>();
  const mode = useWatch({ control, name: "eventTypesMode" });
  const eventTypes = useWatch({ control, name: "eventTypes" });
  const error = formState.errors.eventTypes?.message;
  const statusId = useId();
  const sendingSummary =
    eventTypes.length > 0
      ? `Sending ${eventTypes.length} of ${unkeyAuditLogEvents.options.length} event types.`
      : null;
  const status = error ?? sendingSummary;

  return (
    <fieldset className="flex flex-col gap-1.5">
      <legend className="text-[13px] text-gray-11">Event types</legend>
      <span className="text-xs text-gray-9">
        Choose which audit events to send.{" "}
        <a
          href="https://www.unkey.com/docs/audit-log/types"
          target="_blank"
          rel="noopener noreferrer"
          className="underline underline-offset-2"
        >
          View event types
        </a>
      </span>
      <RadioGroup
        aria-label="Event type scope"
        value={mode}
        onValueChange={(next) =>
          setValue("eventTypesMode", next, { shouldValidate: true, shouldDirty: true })
        }
        className="mt-1.5 grid gap-2 sm:grid-cols-2"
      >
        {eventTypeModes.map((option) => (
          <Radio.Root
            key={option.id}
            value={option.id}
            className="group flex items-center gap-3 rounded-lg border border-grayA-4 px-3 py-2.5 transition-colors duration-150 ease-out focus:outline-hidden focus-visible:ring-2 focus-visible:ring-accent-7 data-checked:border-grayA-8 data-checked:bg-grayA-2"
          >
            <span className="flex size-4 shrink-0 items-center justify-center rounded-full border border-gray-7 transition-colors duration-150 ease-out group-data-checked:border-accent-12">
              <Radio.Indicator className="size-2 rounded-full bg-accent-12" />
            </span>
            <span className="text-[13px] text-accent-12">{option.title}</span>
          </Radio.Root>
        ))}
      </RadioGroup>

      {mode === "specific" ? (
        <div className="mt-1.5 flex flex-col gap-1.5 duration-200 ease-out animate-in fade-in motion-reduce:animate-none">
          <Controller
            control={control}
            name="eventTypes"
            render={({ field }) => (
              <ChoiceMultibox
                {...field}
                options={unkeyAuditLogEvents.options}
                searchLabel="Search event types"
                placeholder="Choose event types"
                emptyMessage="No event types found."
                invalid={Boolean(error)}
                describedBy={status ? statusId : undefined}
              />
            )}
          />
          {status ? (
            <span
              id={statusId}
              role={error ? "alert" : undefined}
              className={cn("text-xs", error ? "text-error-11" : "text-gray-9")}
            >
              {status}
            </span>
          ) : null}
        </div>
      ) : null}
    </fieldset>
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

function VerificationKeyspacesField() {
  const { control } = useFormContext<DrainFormValues>();
  const {
    data: keyspaces = {},
    isLoading,
    error,
  } = trpc.deploy.environmentSettings.getAvailableKeyspaces.useQuery();
  return (
    <Controller
      control={control}
      name="keySpaceIds"
      render={({ field }) => (
        <FilterChoices
          {...field}
          options={Object.keys(keyspaces)}
          label="Keyspaces"
          description="Choose which keyspaces to send verifications from. Leave empty to send all keyspaces."
          searchLabel="Search keyspaces"
          placeholder="All keyspaces"
          emptyMessage={
            error
              ? "Unable to load keyspaces."
              : isLoading
                ? "Loading keyspaces…"
                : "No keyspaces found."
          }
          getLabel={(id) => (keyspaces[id] ? `${keyspaces[id].api.name} (${id})` : id)}
        />
      )}
    />
  );
}

type ChoiceMultiboxProps = {
  value: string[];
  onChange: (value: string[]) => void;
  onBlur: () => void;
  options: readonly string[];
  searchLabel: string;
  placeholder: string;
  emptyMessage: string;
  getLabel?: (choice: string) => string;
  invalid?: boolean;
  describedBy?: string;
};

function ChoiceMultibox({
  value,
  onChange,
  onBlur,
  options,
  searchLabel,
  placeholder,
  emptyMessage,
  getLabel = (choice) => choice || "Unspecified",
  invalid,
  describedBy,
}: ChoiceMultiboxProps) {
  const anchor = useMultiboxAnchor();
  const choices = Array.from(new Set([...options, ...value]));

  return (
    <Multibox items={choices} value={value} onValueChange={onChange} itemToStringLabel={getLabel}>
      <MultiboxChips ref={anchor}>
        {value.map((choice) => (
          <MultiboxChip key={choice}>
            <span className="font-mono">{getLabel(choice)}</span>
            <MultiboxChipRemove />
          </MultiboxChip>
        ))}
        <MultiboxInput
          aria-label={searchLabel}
          aria-invalid={invalid}
          aria-describedby={describedBy}
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
              <span className="font-mono text-xs">{getLabel(choice)}</span>
            </MultiboxItem>
          )}
        </MultiboxList>
      </MultiboxContent>
    </Multibox>
  );
}

function FilterChoices({
  label,
  description,
  ...choices
}: ChoiceMultiboxProps & {
  label: string;
  description: ReactNode;
}) {
  return (
    <fieldset className="flex flex-col gap-1.5">
      <legend className="text-[13px] text-gray-11">{label}</legend>
      <span className="text-xs text-gray-9">{description}</span>
      <div className="mt-1.5">
        <ChoiceMultibox {...choices} />
      </div>
    </fieldset>
  );
}

export function HeaderFields() {
  const { control, register, formState } = useFormContext<DrainFormValues>();
  const { fields, append, remove } = useFieldArray({
    control,
    name: "headers",
  });
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
