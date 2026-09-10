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
            { value: "gateway_requests", label: "Gateway HTTP requests" },
            { value: "runtime_logs", label: "Runtime logs" },
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
    .with("gateway_requests", () => (
      <>
        <GatewayResourcesFields />
        <GatewayStatusesField />
      </>
    ))
    .with("runtime_logs", () => <RuntimeFields />)
    .exhaustive();
}

function GatewayResourcesFields() {
  return (
    <ResourceFields projectField="projectIds" appField="appIds" environmentField="environmentIds" />
  );
}

function RuntimeFields() {
  const { control } = useFormContext<DrainFormValues>();
  return (
    <>
      <ResourceFields
        projectField="runtimeProjectIds"
        appField="runtimeAppIds"
        environmentField="runtimeEnvironmentIds"
      />
      <Controller
        control={control}
        name="severities"
        render={({ field }) => (
          <FilterChoices
            {...field}
            options={["error", "warn", "info", "debug"]}
            label="Severity"
            description="Choose exact severities. Leave empty to send all severities."
            searchLabel="Search severities"
            placeholder="All severities"
            emptyMessage="No severities found."
          />
        )}
      />
    </>
  );
}

function ResourceFields({
  projectField,
  appField,
  environmentField,
}: {
  projectField: "projectIds" | "runtimeProjectIds";
  appField: "appIds" | "runtimeAppIds";
  environmentField: "environmentIds" | "runtimeEnvironmentIds";
}) {
  const { control, getValues, setValue } = useFormContext<DrainFormValues>();
  const projectIds = useWatch({ control, name: projectField });
  const projects = trpc.deploy.project.list.useQuery();
  const environments = trpc.deploy.environment.listAll.useQuery();
  const projectLabels = new Map(projects.data?.map((project) => [project.id, project.name]));
  const appLabels = new Map(
    projects.data?.flatMap((project) =>
      project.apps.map((app) => [app.id, `${project.name} / ${app.name}`] as const),
    ),
  );
  const choices = [
    {
      name: projectField,
      label: "Projects",
      searchLabel: "Search projects",
      placeholder: "All projects",
      labels: projectLabels,
      loading: projects.isLoading,
      error: projects.error,
    },
    {
      name: appField,
      label: "Apps",
      searchLabel: "Search apps",
      placeholder: "All apps",
      labels: new Map(
        projects.data
          ?.filter((project) => projectIds.length === 0 || projectIds.includes(project.id))
          .flatMap((project) =>
            project.apps.map((app) => [app.id, `${project.name} / ${app.name}`] as const),
          ),
      ),
      loading: projects.isLoading,
      error: projects.error,
    },
    {
      name: environmentField,
      label: "Environments",
      searchLabel: "Search environments",
      placeholder: "All environments",
      labels: new Map(
        environments.data?.map((environment) => [
          environment.id,
          `${appLabels.get(environment.appId) ?? environment.appId} / ${environment.name}`,
        ]),
      ),
      loading: environments.isLoading,
      error: environments.error,
    },
  ] as const;
  return (
    <>
      <p className="text-xs text-gray-9">
        Logs must match each selected filter. Leave a filter empty to send all values.
      </p>
      {choices.map((choice) => (
        <Controller
          key={choice.name}
          control={control}
          name={choice.name}
          render={({ field }) => (
            <FilterChoices
              {...field}
              onChange={(values) => {
                field.onChange(values);
                if (choice.name === projectField && values.length > 0 && projects.data) {
                  const appIds = new Set(
                    projects.data
                      .filter((project) => values.includes(project.id))
                      .flatMap((project) => project.apps.map((app) => app.id)),
                  );
                  setValue(
                    appField,
                    getValues(appField).filter((id) => appIds.has(id)),
                    { shouldDirty: true, shouldValidate: true },
                  );
                }
              }}
              options={[...choice.labels.keys()]}
              label={choice.label}
              searchLabel={choice.searchLabel}
              placeholder={choice.placeholder}
              emptyMessage={
                choice.error
                  ? `Unable to load ${choice.label.toLowerCase()}.`
                  : choice.loading
                    ? "Loading…"
                    : "No matches found."
              }
              getLabel={(id) => (choice.labels.has(id) ? `${choice.labels.get(id)} (${id})` : id)}
            />
          )}
        />
      ))}
    </>
  );
}

function GatewayStatusesField() {
  const { control } = useFormContext<DrainFormValues>();
  return (
    <Controller
      control={control}
      name="statusClasses"
      render={({ field }) => (
        <FilterChoices
          value={field.value.map(String)}
          onChange={(values) => field.onChange(values.map(Number))}
          onBlur={field.onBlur}
          options={["2", "3", "4", "5"]}
          getLabel={(value) => `${value}xx`}
          label="HTTP statuses"
          description="Choose status classes. Leave empty to send all statuses."
          searchLabel="Search HTTP statuses"
          placeholder="All HTTP statuses"
          emptyMessage="No HTTP statuses found."
        />
      )}
    />
  );
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
  getLabel = (choice) => choice || "Unspecified",
}: {
  value: string[];
  onChange: (value: string[]) => void;
  onBlur: () => void;
  options: readonly string[];
  label: string;
  description?: ReactNode;
  searchLabel: string;
  placeholder: string;
  emptyMessage: string;
  getLabel?: (choice: string) => string;
}) {
  const anchor = useMultiboxAnchor();
  const choices = Array.from(new Set([...options, ...value]));

  return (
    <fieldset className="flex flex-col gap-1.5">
      <legend className="text-[13px] text-gray-11">{label}</legend>
      {description ? <span className="text-xs text-gray-9">{description}</span> : null}
      <Multibox items={choices} value={value} onValueChange={onChange} itemToStringLabel={getLabel}>
        <MultiboxChips ref={anchor} className="mt-1.5">
          {value.map((choice) => (
            <MultiboxChip key={choice}>
              <span className="font-mono">{getLabel(choice)}</span>
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
                <span className="font-mono text-xs">{getLabel(choice)}</span>
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
