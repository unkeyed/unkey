"use client";

import type { Policy } from "@/lib/collections/deploy/policies.schema";
import { zodResolver } from "@hookform/resolvers/zod";
import { toast } from "@unkey/ui";
import { useState } from "react";
import { useForm, useFormState } from "react-hook-form";
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
  onUpdate: (policy: Policy, row: MergedPolicy) => void;
  onToggleEnv: (key: string, env: Env) => void;
  onAddToEnv: (key: string, env: Env) => void;
};

/** `session` changes on every open, so each edit starts from a fresh form. */
export function EditPolicyPanel({
  editing,
  ...props
}: EditPolicyPanelProps & { editing: { session: number } | null }) {
  if (editing === null) {
    return null;
  }
  return <EditPolicyForm key={editing.session} {...props} />;
}

function EditPolicyForm({
  row,
  envs,
  isOpen,
  onClose,
  existingMatchKeys,
  onUpdate,
  onToggleEnv,
  onAddToEnv,
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
    if (row) {
      onUpdate(toPolicy(values, policy.id), row);
    } else {
      toast.error("Couldn't save. This policy no longer exists.");
    }
    onClose();
  };

  return (
    <PolicyForm
      title="Edit policy"
      submitLabel="Save changes"
      isOpen={isOpen}
      onClose={onClose}
      form={form}
      onSubmit={onSubmit}
    >
      <div className="flex flex-col gap-1.5">
        <span className="text-sm font-medium text-gray-12">Environments</span>
        <PolicyEnvironmentToggles
          policy={row ?? opened}
          envs={envs}
          locked={isDirty}
          onToggle={onToggleEnv}
          onAdd={onAddToEnv}
        />
      </div>
    </PolicyForm>
  );
}
