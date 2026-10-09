"use client";

import { EnvironmentLabel } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/apps/_components/environment-label";
import { ENVIRONMENT_KINDS } from "@/lib/collections/deploy/environments";
import type { PolicyInput } from "@/lib/collections/deploy/policies.schema";
import { zodResolver } from "@hookform/resolvers/zod";
import { IconChevronDownOutline18 } from "@unkey/icons";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@unkey/ui";
import { type ReactNode, useId, useState } from "react";
import { useForm } from "react-hook-form";
import {
  type PolicyFormValues,
  type PolicyType,
  getDefaultValues,
  policyFormSchema,
  toPolicy,
} from "../../policy-kinds";
import type { Env, PolicyEnvs } from "../list/merge";
import { duplicateNameError } from "./duplicate-name";
import { PolicyForm } from "./policy-form";

type EnvironmentTarget = "all" | Env;

type EnvironmentOption = { value: EnvironmentTarget; label: string; content: ReactNode };

const ALL_OPTION: EnvironmentOption = {
  value: "all",
  label: "All environments",
  content: "All environments",
};

export function AddPolicyPanel({
  envs,
  isOpen,
  onClose,
  existingMatchKeys,
  onSave,
}: {
  envs: PolicyEnvs;
  isOpen: boolean;
  onClose: () => void;
  existingMatchKeys: string[];
  onSave: (policy: PolicyInput, envs: readonly Env[]) => Promise<boolean>;
}) {
  const environmentSelectId = useId();
  const [target, setTarget] = useState<EnvironmentTarget>("all");
  const environmentOptions: EnvironmentOption[] = [
    ALL_OPTION,
    ...ENVIRONMENT_KINDS.map((kind) => ({
      value: kind,
      label: envs[kind].slug,
      content: (
        <EnvironmentLabel
          environment={{ kind, slug: envs[kind].slug }}
          className="text-sm text-gray-12"
        />
      ),
    })),
  ];

  const selected = environmentOptions.find((option) => option.value === target) ?? ALL_OPTION;

  const form = useForm<PolicyFormValues>({
    resolver: zodResolver(policyFormSchema),
    defaultValues: getDefaultValues("keyauth"),
  });

  // Reset to the new type's defaults so its fields don't leak between union
  // branches, but keep the work shared by all types. The original defaults
  // stay so isDirty still sees a type switch or carried-over name.
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
    const duplicate = duplicateNameError(values, existingMatchKeys, null);
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
      <div className="flex flex-col gap-1.5">
        <label htmlFor={environmentSelectId} className="text-sm font-medium text-gray-12">
          Environment
        </label>
        <Select
          value={target}
          onValueChange={(value) => {
            if (value !== null) {
              setTarget(value);
            }
          }}
          items={environmentOptions.map(({ value, label }) => ({ value, label }))}
        >
          <SelectTrigger
            id={environmentSelectId}
            wrapperClassName="w-60"
            rightIcon={<IconChevronDownOutline18 className="size-3.5 absolute right-2" />}
          >
            <SelectValue placeholder="Select environment">{selected.content}</SelectValue>
          </SelectTrigger>
          <SelectContent>
            {environmentOptions.map((option) => (
              <SelectItem key={option.value} value={option.value}>
                {option.content}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>
    </PolicyForm>
  );
}
