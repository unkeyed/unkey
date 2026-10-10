"use client";

import type { PolicyInput } from "@/lib/collections/deploy/policies.schema";
import { zodResolver } from "@hookform/resolvers/zod";
import { useState } from "react";
import { useForm, useFormState } from "react-hook-form";
import { duplicateNameError } from "../../model/duplicate-name";
import type { Env, MergedPolicy, PolicyEnvs } from "../../model/merge";
import { type PolicyFormValues, fromPolicy, policyFormSchema, toPolicy } from "../../policy-kinds";
import { PolicyEnvironmentToggles } from "../env-toggles";
import { PolicyForm } from "./policy-form";

type EditPolicyPanelProps = {
  envs: PolicyEnvs;
  isOpen: boolean;
  onClose: () => void;
  otherMatchKeys: string[];
  row: MergedPolicy | undefined;
  locked: boolean;
  onUpdate: (key: string, policy: PolicyInput) => Promise<boolean>;
  onSetEnabled: (key: string, env: Env, enabled: boolean) => Promise<boolean>;
};

export function EditPolicyPanel({
  row,
  locked,
  envs,
  isOpen,
  onClose,
  otherMatchKeys,
  onUpdate,
  onSetEnabled,
}: EditPolicyPanelProps) {
  const [rowAtOpen] = useState(row);
  const policy = rowAtOpen?.production ?? rowAtOpen?.preview;
  const form = useForm<PolicyFormValues>({
    resolver: zodResolver(policyFormSchema),
    defaultValues: policy ? fromPolicy(policy) : undefined,
  });
  const { isDirty } = useFormState({ control: form.control });

  if (!rowAtOpen || !policy) {
    return null;
  }

  const onSubmit = async (values: PolicyFormValues) => {
    const duplicate = duplicateNameError(values, otherMatchKeys);
    if (duplicate) {
      form.setError("name", { type: "manual", message: duplicate });
      return;
    }
    if (await onUpdate(rowAtOpen.key, toPolicy(values))) {
      onClose();
    }
  };

  const live = row ?? rowAtOpen;

  return (
    <PolicyForm
      title="Edit policy"
      submitLabel="Save"
      isOpen={isOpen}
      onClose={onClose}
      form={form}
      onSubmit={onSubmit}
      submitDisabled={locked}
    >
      <div className="flex flex-col gap-1.5">
        <span className="text-sm font-medium text-gray-12">Environments</span>
        <PolicyEnvironmentToggles
          policy={live}
          envs={envs}
          dirty={isDirty}
          locked={locked}
          onToggle={(env, enabled) => void onSetEnabled(live.key, env, enabled)}
        />
      </div>
    </PolicyForm>
  );
}
