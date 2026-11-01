"use client";

import { EnvironmentLabel } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/apps/_components/environment-label";
import type { PolicyInput } from "@/lib/collections/deploy/policies.schema";
import { ENVIRONMENT_KINDS } from "@/lib/deploy-data";
import { zodResolver } from "@hookform/resolvers/zod";
import { FormSelect, type FormSelectOption } from "@unkey/ui";
import { useState } from "react";
import { useForm } from "react-hook-form";
import { duplicateNameError } from "../../model/duplicate-name";
import type { Env, PolicyEnvs } from "../../model/merge";
import {
  type PolicyFormValues,
  type PolicyType,
  getDefaultValues,
  policyFormSchema,
  toPolicy,
} from "../../policy-kinds";
import { PolicyForm } from "./policy-form";

type EnvironmentTarget = "all" | Env;

export function AddPolicyPanel({
  envs,
  isOpen,
  onClose,
  otherMatchKeys,
  onSave,
}: {
  envs: PolicyEnvs;
  isOpen: boolean;
  onClose: () => void;
  otherMatchKeys: string[];
  onSave: (policy: PolicyInput, envs: readonly Env[]) => Promise<boolean>;
}) {
  const [target, setTarget] = useState<EnvironmentTarget>("all");
  const environmentOptions: FormSelectOption<EnvironmentTarget>[] = [
    { value: "all", label: "All environments" },
    ...ENVIRONMENT_KINDS.map((kind) => ({
      value: kind,
      label: (
        <EnvironmentLabel
          environment={{ kind, slug: envs[kind].slug }}
          className="text-sm text-gray-12"
        />
      ),
    })),
  ];

  const form = useForm<PolicyFormValues>({
    resolver: zodResolver(policyFormSchema),
    defaultValues: getDefaultValues("keyauth"),
  });

  const changeType = (type: PolicyType) => {
    form.reset(
      {
        ...getDefaultValues(type),
        name: form.getValues("name"),
        matchConditions: form.getValues("matchConditions"),
      },
      { keepDefaultValues: true },
    );
  };

  const onSubmit = async (values: PolicyFormValues) => {
    const duplicate = duplicateNameError(values, otherMatchKeys);
    if (duplicate) {
      form.setError("name", { type: "manual", message: duplicate });
      return;
    }

    const targets = ENVIRONMENT_KINDS.filter((env) => target === "all" || target === env);
    const saved = await onSave(toPolicy(values), targets);
    if (!saved) {
      return;
    }
    onClose();
  };

  return (
    <PolicyForm
      title="Add policy"
      submitLabel="Add policy"
      isOpen={isOpen}
      onClose={onClose}
      form={form}
      onSubmit={onSubmit}
      onTypeChange={changeType}
    >
      <FormSelect
        label="Environment"
        options={environmentOptions}
        value={target}
        onValueChange={setTarget}
        className="w-60"
      />
    </PolicyForm>
  );
}
