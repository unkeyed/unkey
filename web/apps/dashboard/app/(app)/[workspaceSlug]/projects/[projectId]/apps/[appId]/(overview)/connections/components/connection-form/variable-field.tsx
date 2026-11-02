"use client";

import { EnvironmentIcon } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/apps/_components/environment-label";
import { Badge, FormInput } from "@unkey/ui";
import { Controller, useFormContext, useWatch } from "react-hook-form";
import type { ConnectionFormValues } from "../../model/form";
import { connectionHostVariable } from "../../model/name";
import { renameOf } from "../../model/plan";
import type { ListedConnection } from "../../model/target";
import type { ConnectedTarget } from "../../model/target-options";

export function VariableField({ before }: { before: ListedConnection | null }) {
  const { control, trigger } = useFormContext<ConnectionFormValues>();
  const name = useWatch({ control, name: "name" }).trim();
  const renamed = renameOf(before, name) !== null;

  return (
    <div className="flex flex-col gap-2">
      <Controller
        control={control}
        name="name"
        render={({ field, fieldState }) => (
          <FormInput
            label="Name"
            value={field.value}
            onChange={(event) => {
              field.onChange(event);
              if (fieldState.invalid) {
                void trigger("name");
              }
            }}
            onBlur={field.onBlur}
            error={fieldState.error?.message}
            description={
              name ? (
                <>
                  Available in your app as{" "}
                  <code className="font-mono text-gray-12">{connectionHostVariable(name)}</code>.
                </>
              ) : undefined
            }
          />
        )}
      />
      {renamed ? (
        <p className="text-xs text-warning-11">
          Renaming changes the variable. Update your code before you redeploy.
        </p>
      ) : null}
    </div>
  );
}

export function VariableSummary({ connected }: { connected: readonly ConnectedTarget[] }) {
  const { control } = useFormContext<ConnectionFormValues>();
  const name = useWatch({ control, name: "name" }).trim();
  return (
    <>
      <code className="truncate font-mono text-xs">{connectionHostVariable(name)}</code>
      {connected.map(({ kind, label }) => (
        <Badge key={kind} variant="secondary" size="compact" className="shrink-0 rounded-full">
          <EnvironmentIcon kind={kind} className="size-3" />
          {label}
        </Badge>
      ))}
    </>
  );
}
