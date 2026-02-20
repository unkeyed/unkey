"use client";

import type { ComboboxOption } from "@/components/ui/combobox";
import { FormCombobox } from "@/components/ui/form-combobox";
import { collection } from "@/lib/collections";
import { trpc } from "@/lib/trpc/client";
import { zodResolver } from "@hookform/resolvers/zod";
import { eq, useLiveQuery } from "@tanstack/react-db";
import { Key, XMark } from "@unkey/icons";
import { useEffect } from "react";
import { useForm, useWatch } from "react-hook-form";
import { z } from "zod";
import { useEnvironmentId } from "../../environment-provider";
import { FormSettingCard } from "../shared/form-setting-card";

const keyspacesSchema = z.object({
  keyspaces: z.array(z.string()).min(1, "Select at least one region"),
});

type KeyspacesFormValues = z.infer<typeof keyspacesSchema>;

export const Keyspaces = () => {
  const environmentId = useEnvironmentId();

  const { data: settings } = useLiveQuery(
    (q) =>
      q
        .from({ s: collection.environmentSettings })
        .where(({ s }) => eq(s.environmentId, environmentId)),
    [environmentId],
  );

  const { data: availableKeyspaces } =
    trpc.deploy.environmentSettings.getAvailableKeyspaces.useQuery(undefined, {
      enabled: Boolean(environmentId),
    });

  const defaultKeyspaceIds: string[] = [];
  for (const policy of settings?.[0]?.sentinelConfig?.policies ?? []) {
    if (policy.keyauth) {
      defaultKeyspaceIds.push(...policy.keyauth.keySpaceIds);
    }
  }

  return (
    <KeyspacesForm
      environmentId={environmentId}
      defaultKeyspaceIds={defaultKeyspaceIds}
      availableKeyspaces={availableKeyspaces ?? {}}
    />
  );
};

type KeyspacesFormProps = {
  environmentId: string;
  defaultKeyspaceIds: string[];
  availableKeyspaces: Record<string, { id: string; api: { name: string } }>;
};

const KeyspacesForm: React.FC<KeyspacesFormProps> = ({
  environmentId,
  defaultKeyspaceIds,
  availableKeyspaces,
}) => {
  const {
    handleSubmit,
    setValue,
    formState: { isValid, isSubmitting },
    control,
    reset,
  } = useForm<KeyspacesFormValues>({
    resolver: zodResolver(keyspacesSchema),
    mode: "onChange",
    defaultValues: { keyspaces: defaultKeyspaceIds },
  });

  useEffect(() => {
    reset({ keyspaces: defaultKeyspaceIds });
  }, [defaultKeyspaceIds, reset]);

  const currentKeyspaceIds = useWatch({ control, name: "keyspaces" });

  const unselectedKeyspaceIds = Object.keys(availableKeyspaces).filter(
    (r) => !currentKeyspaceIds.includes(r),
  );

  const onSubmit = async (values: KeyspacesFormValues) => {
    collection.environmentSettings.update(environmentId, (draft) => {
      draft.sentinelConfig = {
        policies:
          values.keyspaces.length > 0
            ? [
                {
                  id: "keyauth-policy",
                  name: "API Key Auth",
                  enabled: true,
                  keyauth: { keySpaceIds: values.keyspaces },
                },
              ]
            : [],
      };
    });
  };

  const addKeyspace = (region: string) => {
    if (region && !currentKeyspaceIds.includes(region)) {
      setValue("keyspaces", [...currentKeyspaceIds, region], {
        shouldValidate: true,
      });
    }
  };

  const removeKeyspace = (region: string) => {
    setValue(
      "keyspaces",
      currentKeyspaceIds.filter((r) => r !== region),
      { shouldValidate: true },
    );
  };

  const hasChanges =
    currentKeyspaceIds.length !== defaultKeyspaceIds.length ||
    currentKeyspaceIds.some((r) => !defaultKeyspaceIds.includes(r));

  const displayValue =
    defaultKeyspaceIds.length === 0 ? (
      "No keyspaces selected"
    ) : defaultKeyspaceIds.length <= 2 ? (
      <span className="flex items-center gap-1.5">
        {defaultKeyspaceIds.map((keyspaceId, i) => (
          <span key={keyspaceId} className="flex items-center gap-1.5">
            {i > 0 && <span className="text-grayA-4">|</span>}
            <span className="text-gray-11">
              {availableKeyspaces[keyspaceId]?.api?.name ?? keyspaceId}
            </span>
          </span>
        ))}
      </span>
    ) : (
      <span className="flex items-center gap-1">
        {defaultKeyspaceIds.map((keyspaceId) => (
          <span key={keyspaceId}> {availableKeyspaces[keyspaceId]?.api?.name ?? keyspaceId}</span>
        ))}
      </span>
    );

  const comboboxOptions: ComboboxOption[] = unselectedKeyspaceIds.map((keyspaceId) => ({
    value: keyspaceId,
    searchValue: keyspaceId,
    label: (
      <span className="text-gray-11 text-xs font-mono">
        {availableKeyspaces[keyspaceId]?.api?.name ?? keyspaceId}
      </span>
    ),
  }));

  return (
    <FormSettingCard
      icon={<Key className="text-gray-12" iconSize="xl-medium" />}
      title="Keyspaces"
      description="Enforce key authentication in your sentinel."
      displayValue={displayValue}
      onSubmit={handleSubmit(onSubmit)}
      canSave={isValid && !isSubmitting && hasChanges}
      isSaving={isSubmitting}
    >
      <FormCombobox
        label="Keyspaces"
        description="Sentinels handle auth before the request even reaches your API"
        optional
        className="w-[480px]"
        options={comboboxOptions}
        value=""
        onSelect={addKeyspace}
        placeholder={
          currentKeyspaceIds.length === 0 ? (
            <span className="text-grayA-8 w-full text-left">Select a keyspace</span>
          ) : (
            <div className="w-full flex flex-wrap gap-1.5 py-0.5">
              {currentKeyspaceIds.map((keyspaceId) => (
                <span
                  key={keyspaceId}
                  className="flex items-center gap-1 px-1.5 py-0.5 rounded-md bg-grayA-3 border border-grayA-4 text-xs text-accent-12"
                >
                  {availableKeyspaces[keyspaceId]?.api?.name ?? keyspaceId}
                  {currentKeyspaceIds.length > 1 && (
                    //biome-ignore lint/a11y/useKeyWithClickEvents: we can't use button here otherwise we'll nest two buttons
                    <span
                      onClick={(e) => {
                        e.stopPropagation();
                        removeKeyspace(keyspaceId);
                      }}
                      className="p-0.5 hover:bg-grayA-4 rounded text-grayA-9 hover:text-accent-12 transition-colors"
                    >
                      <XMark iconSize="sm-regular" />
                    </span>
                  )}
                </span>
              ))}
            </div>
          )
        }
        searchPlaceholder="Search keyspaces..."
        emptyMessage={<div className="mt-2">No keyspaces available.</div>}
      />
    </FormSettingCard>
  );
};
