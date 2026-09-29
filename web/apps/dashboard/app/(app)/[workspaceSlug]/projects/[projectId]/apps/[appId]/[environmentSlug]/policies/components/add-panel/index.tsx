"use client";

import { type Policy, policyMatchKey } from "@/lib/collections/deploy/policies.schema";
import { zodResolver } from "@hookform/resolvers/zod";
import { match } from "@unkey/match";
import { Button, Checkbox, FormInput, FormSelect } from "@unkey/ui";
import { useId } from "react";
import { Controller, useForm, useWatch } from "react-hook-form";
import { FirewallFields, FirewallPolicySummary } from "./forms/firewall-fields";
import { KeyAuthFields, KeyauthPolicySummary } from "./forms/keyauth-fields";
import { LoggingFields, LoggingPolicySummary } from "./forms/logging-fields";
import { OpenApiFields, OpenApiPolicySummary } from "./forms/openapi-fields";
import { RateLimitFields, RatelimitPolicySummary } from "./forms/ratelimit-fields";
import { DocsLink } from "./forms/summary-helpers";
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
  fromPolicy,
  getDefaultValues,
  policyFormSchema,
  toPolicy,
} from "./schema";

type CommonProps = {
  otherEnvironmentSlug: string;
  isOpen: boolean;
  onClose: () => void;
  /**
   * `policyMatchKey` of every policy on the page. `mergePolicies` pairs the
   * two environment copies of a policy on that match key, so a second policy
   * carrying one is indistinguishable from the first and is rejected here.
   */
  existingMatchKeys: string[];
};

type OnSave = (policy: Policy, applyToOtherEnvironment: boolean) => void;

type AddProps = CommonProps & {
  mode: "add";
  onSave: OnSave;
};

type EditProps = CommonProps & {
  mode: "edit";
  initialPolicy: Policy;
  initialApplyToOther: boolean;
  onSave: OnSave;
};

export type PolicyPanelProps = AddProps | EditProps;

export function PolicyPanel(props: PolicyPanelProps) {
  const { otherEnvironmentSlug, isOpen, onClose, existingMatchKeys } = props;
  const isEdit = props.mode === "edit";

  const form = useForm<PolicyFormValues>({
    resolver: zodResolver(policyFormSchema),
    defaultValues: isEdit
      ? fromPolicy(props.initialPolicy, props.initialApplyToOther)
      : getDefaultValues("keyauth"),
  });
  const { control } = form;
  const policyType = useWatch({ control, name: "type" });
  const applyToOtherId = useId();

  const onSubmit = (values: PolicyFormValues) => {
    const nextMatchKey = policyMatchKey(values.type, values.name);
    const currentMatchKey = isEdit
      ? policyMatchKey(props.initialPolicy.type, props.initialPolicy.name)
      : null;
    if (nextMatchKey !== currentMatchKey && existingMatchKeys.includes(nextMatchKey)) {
      const label =
        POLICY_TYPE_OPTIONS.find((option) => option.value === values.type)?.label ?? values.type;
      form.setError("name", {
        type: "manual",
        message: `A ${label} policy named "${values.name}" already exists. Use the + on its row to add it to another environment.`,
      });
      return;
    }

    const id = props.mode === "edit" ? props.initialPolicy.id : undefined;
    props.onSave(toPolicy(values, id), values.applyToOtherEnvironment);
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
          {isEdit ? "Update this gateway policy." : "Configure and add a new gateway policy."}
          <DocsLink href="https://www.unkey.com/docs/platform/gateway/policies/overview">
            <span className="text-[13px]">See docs for more</span>
          </DocsLink>
        </div>
      }
      isOpen={isOpen}
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
              requirement="required"
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
                // discriminated union. Shared fields (name, the other
                // environment choice, matchConditions) are preserved so the
                // user doesn't lose work when they switch types.
                if (isEdit) {
                  return;
                }
                const defaults = getDefaultValues(next as PolicyType);
                form.reset({
                  ...defaults,
                  name: form.getValues("name"),
                  applyToOtherEnvironment: form.getValues("applyToOtherEnvironment"),
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
          summary={match(policyType)
            .with("keyauth", () => <KeyauthPolicySummary />)
            .with("ratelimit", () => <RatelimitPolicySummary />)
            .with("firewall", () => <FirewallPolicySummary />)
            .with("openapi", () => <OpenApiPolicySummary />)
            .with("logging", () => <LoggingPolicySummary />)
            .exhaustive()}
          catchAll
        >
          {match(policyType)
            .with("keyauth", () => <KeyAuthFields />)
            .with("ratelimit", () => <RateLimitFields />)
            .with("firewall", () => <FirewallFields />)
            .with("openapi", () => <OpenApiFields />)
            .with("logging", () => <LoggingFields />)
            .exhaustive()}
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
        <div className="border-t">
          <div className="px-6 py-6">
            <Controller
              control={control}
              name="applyToOtherEnvironment"
              render={({ field }) => (
                <label htmlFor={applyToOtherId} className="flex items-start gap-3 cursor-pointer">
                  <Checkbox
                    id={applyToOtherId}
                    checked={field.value}
                    onCheckedChange={(checked) => field.onChange(checked === true)}
                    className="mt-0.5 size-4 rounded-sm [&_svg]:size-3"
                  />
                  <span className="flex flex-col gap-0.5">
                    <span className="text-[13px] text-gray-12">
                      Also apply to <span className="capitalize">{otherEnvironmentSlug}</span>
                    </span>
                    <span className="text-xs text-gray-9">
                      Adds or enables the same policy there. Left unchecked, a copy there is
                      switched off.
                    </span>
                  </span>
                </label>
              )}
            />
          </div>
        </div>

        <div className="border-t bg-raised px-6 py-5 flex items-center justify-end">
          <Button type="submit" variant="primary" size="md" className="px-3">
            {isEdit ? "Save Changes" : "Add Policy"}
          </Button>
        </div>
      </PolicyForm.Footer>
    </PolicyForm>
  );
}
