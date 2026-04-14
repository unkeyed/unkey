"use client";

import type { SentinelPolicy } from "@/lib/collections/deploy/sentinel-policies.schema";
import { zodResolver } from "@hookform/resolvers/zod";
import { Button, FormInput, FormSelect } from "@unkey/ui";
import { Controller, useForm, useWatch } from "react-hook-form";
import { FirewallFields, FirewallSummary } from "./forms/firewall-fields";
import { KeyAuthFields, PolicySummary } from "./forms/keyauth-fields";
import {
  MatchConditionEditorBody,
  MatchConditionsClearAll,
  MatchConditionsSummary,
} from "./match-condition-editor";
import { PolicyForm } from "./policy-form";
import {
  POLICY_TYPE_OPTIONS,
  type PolicyFormValues,
  type PolicyType,
  fromSentinelPolicy,
  getDefaultValues,
  policyFormSchema,
  toSentinelPolicy,
} from "./schema";

type CommonProps = {
  envASlug: string;
  envBSlug: string;
  isOpen: boolean;
  topOffset: number;
  onClose: () => void;
};

type AddProps = CommonProps & {
  mode: "add";
  onSave: (prodPolicy: SentinelPolicy | null, previewPolicy: SentinelPolicy | null) => void;
};

type EditProps = CommonProps & {
  mode: "edit";
  initialPolicy: SentinelPolicy;
  initialEnvironmentId: string;
  onSave: (prodPolicy: SentinelPolicy | null, previewPolicy: SentinelPolicy | null) => void;
};

export type SentinelPolicyPanelProps = AddProps | EditProps;

export function SentinelPolicyPanel(props: SentinelPolicyPanelProps) {
  const { envASlug, envBSlug, isOpen, topOffset, onClose } = props;
  const isEdit = props.mode === "edit";

  const envOptions = [
    { value: "__all__", label: "All Environments" },
    { value: envASlug, label: envASlug },
    { value: envBSlug, label: envBSlug },
  ];

  const form = useForm<PolicyFormValues>({
    resolver: zodResolver(policyFormSchema),
    defaultValues: isEdit
      ? fromSentinelPolicy(props.initialPolicy, props.initialEnvironmentId)
      : getDefaultValues("keyauth"),
  });
  const { control } = form;
  const policyType = useWatch({ control, name: "type" }) as PolicyType;

  const onSubmit = (values: PolicyFormValues) => {
    const id = props.mode === "edit" ? props.initialPolicy.id : undefined;
    const policy = toSentinelPolicy(values, id);
    const prodPolicy =
      values.environmentId === "__all__" || values.environmentId === envASlug
        ? { ...policy, enabled: true }
        : null;
    const previewPolicy =
      values.environmentId === "__all__" || values.environmentId === envBSlug
        ? { ...policy, enabled: true }
        : null;

    props.onSave(prodPolicy, previewPolicy);
    onClose();
    if (props.mode === "add") {
      form.reset(getDefaultValues("keyauth"));
    }
  };

  return (
    <PolicyForm
      title={isEdit ? "Edit Policy" : "Add Policy"}
      description={
        <div className="flex gap-2 items-center">
          {isEdit ? "Update this sentinel policy." : "Configure and add a new sentinel policy."}
          <a
            href="https://www.unkey.com/docs/platform/sentinel/policies/overview"
            target="_blank"
            rel="noopener noreferrer"
          >
            <span className="font-medium text-gray-12 underline underline-offset-2 decoration-grayA-6 group-hover:decoration-gray-12 transition-colors decoration-dotted">
              See docs for more
            </span>
          </a>
        </div>
      }
      isOpen={isOpen}
      topOffset={topOffset}
      onClose={onClose}
      form={form}
      onSubmit={onSubmit}
    >
      <PolicyForm.Fields>
        <Controller
          control={control}
          name="name"
          render={({ field, fieldState }) => (
            <FormInput
              label="Name"
              required
              descriptionPosition="label"
              placeholder="e.g. API Key Auth, Rate Limit Public"
              description="A descriptive name to identify this policy."
              value={field.value}
              onChange={field.onChange}
              error={fieldState.error?.message}
            />
          )}
        />

        <Controller
          control={control}
          name="type"
          render={({ field }) => (
            <FormSelect
              label="Type"
              options={POLICY_TYPE_OPTIONS}
              value={field.value}
              onValueChange={(next) => {
                // Reset the form to the defaults of the newly-chosen type so
                // type-specific fields don't leak between branches of the
                // discriminated union. Shared fields (name, environmentId,
                // matchConditions) are preserved so the user doesn't lose
                // work when they switch types after starting to configure.
                if (isEdit) {
                  return;
                }
                const defaults = getDefaultValues(next as PolicyType);
                form.reset({
                  ...defaults,
                  name: form.getValues("name"),
                  environmentId: form.getValues("environmentId"),
                  matchConditions: form.getValues("matchConditions"),
                });
                field.onChange(next);
              }}
              disabled={isEdit}
              description="The kind of protection this policy enforces."
              descriptionPosition="label"
              triggerClassName="capitalize"
            />
          )}
        />
      </PolicyForm.Fields>

      <PolicyForm.Accordion defaultExpanded="config">
        <PolicyForm.Section
          id="config"
          label="Policy Configuration"
          summary={policyType === "firewall" ? <FirewallSummary /> : <PolicySummary />}
          catchAll
        >
          {policyType === "firewall" ? <FirewallFields /> : <KeyAuthFields />}
        </PolicyForm.Section>
        <PolicyForm.Section
          id="matchConditions"
          label="Match Conditions"
          summary={<MatchConditionsSummary />}
          fields={["matchConditions"]}
          tooltipContent={
            <span>
              All conditions must match (<span className="text-gray-12 font-medium">AND</span>{" "}
              logic).
            </span>
          }
          collapsedAction={<MatchConditionsClearAll />}
        >
          <MatchConditionEditorBody />
        </PolicyForm.Section>
      </PolicyForm.Accordion>
      <PolicyForm.Footer>
        <div className="border-t border-grayA-4">
          <div className="px-8 py-6">
            <Controller
              control={control}
              name="environmentId"
              render={({ field }) => (
                <FormSelect
                  label="Environment"
                  options={envOptions}
                  value={field.value}
                  onValueChange={field.onChange}
                  description="Which environments this policy will be added to."
                  descriptionPosition="label"
                  triggerClassName="capitalize"
                />
              )}
            />
          </div>
        </div>

        <div className="border-t border-gray-4 bg-white dark:bg-black px-8 py-5 flex items-center justify-end">
          <Button type="submit" variant="primary" size="md" className="px-3">
            {isEdit ? "Save Changes" : "Add Policy"}
          </Button>
        </div>
      </PolicyForm.Footer>
    </PolicyForm>
  );
}
