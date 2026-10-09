"use client";

import type { PolicyInput } from "@/lib/collections/deploy/policies.schema";
import { zodResolver } from "@hookform/resolvers/zod";
import { useState } from "react";
import { useForm, useFormState } from "react-hook-form";
import type { PolicySwitches } from "../../hooks/policy-switches";
import { type PolicyFormValues, fromPolicy, policyFormSchema, toPolicy } from "../../policy-kinds";
import { PolicyEnvironmentToggles } from "../env-toggles";
import type { Env, MergedPolicy, PolicyEnvs, PolicyRowKey } from "../list/merge";
import { duplicateNameError } from "./duplicate-name";
import { PolicyForm } from "./policy-form";

type EditPolicyPanelProps = {
  envs: PolicyEnvs;
  isOpen: boolean;
  onClose: () => void;
  existingMatchKeys: string[];
  row: MergedPolicy | undefined;
  switchesOf: (policy: MergedPolicy) => PolicySwitches;
  onUpdate: (key: PolicyRowKey, policy: PolicyInput) => void;
  onToggleEnv: (key: PolicyRowKey, env: Env) => void;
};

export function EditPolicyPanel({
  row,
  switchesOf,
  envs,
  isOpen,
  onClose,
  existingMatchKeys,
  onUpdate,
  onToggleEnv,
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

  const onSubmit = (values: PolicyFormValues) => {
    const duplicate = duplicateNameError(values, existingMatchKeys, rowAtOpen);
    if (duplicate) {
      form.setError("name", { type: "manual", message: duplicate });
      return;
    }
    onUpdate(rowAtOpen.key, toPolicy(values));
    onClose();
  };

  return (
    <PolicyForm
      title="Edit policy"
      submitLabel="Save"
      isOpen={isOpen}
      onClose={onClose}
      form={form}
      onSubmit={onSubmit}
    >
      <div className="flex flex-col gap-1.5">
        <span className="text-sm font-medium text-gray-12">Environments</span>
        <PolicyEnvironmentToggles
          policy={row ?? rowAtOpen}
          switches={switchesOf(row ?? rowAtOpen)}
          envs={envs}
          dirty={isDirty}
          onToggle={onToggleEnv}
        />
      </div>
    </PolicyForm>
  );
}
