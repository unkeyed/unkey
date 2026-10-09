"use client";

import type { PolicyInput } from "@/lib/collections/deploy/policies.schema";
import { zodResolver } from "@hookform/resolvers/zod";
import { useState } from "react";
import { useForm, useFormState } from "react-hook-form";
import type { PolicySwitches } from "../../hooks/policy-switches";
import { type PolicyFormValues, fromPolicy, policyFormSchema, toPolicy } from "../../policy-kinds";
import { PolicyEnvironmentToggles } from "../env-toggles";
import type { Env, MergedPolicy, PolicyEnvs } from "../list/merge";
import { duplicateNameError } from "./duplicate-name";
import { PolicyForm } from "./policy-form";

type EditPolicyPanelProps = {
  envs: PolicyEnvs;
  isOpen: boolean;
  onClose: () => void;
  existingMatchKeys: string[];
  row: MergedPolicy | undefined;
  switchesOf: (policy: MergedPolicy) => PolicySwitches;
  onUpdate: (key: string, policy: PolicyInput) => void;
  onToggleEnv: (key: string, env: Env) => void;
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
  // Renaming the policy changes its row key, so the live row is gone while the
  // panel slides out after a save. The row it opened with fills in until then.
  const [opened] = useState(row);
  const policy = opened?.production ?? opened?.preview;
  const form = useForm<PolicyFormValues>({
    resolver: zodResolver(policyFormSchema),
    defaultValues: policy ? fromPolicy(policy) : undefined,
  });
  const { isDirty } = useFormState({ control: form.control });

  if (!opened || !policy) {
    return null;
  }

  const onSubmit = (values: PolicyFormValues) => {
    const duplicate = duplicateNameError(values, existingMatchKeys, opened);
    if (duplicate) {
      form.setError("name", { type: "manual", message: duplicate });
      return;
    }
    onUpdate(opened.key, toPolicy(values));
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
          policy={row ?? opened}
          switches={switchesOf(row ?? opened)}
          envs={envs}
          dirty={isDirty}
          onToggle={onToggleEnv}
        />
      </div>
    </PolicyForm>
  );
}
