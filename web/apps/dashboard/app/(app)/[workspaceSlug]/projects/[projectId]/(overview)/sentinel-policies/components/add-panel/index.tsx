"use client";

import type { SentinelPolicy } from "@/lib/collections/deploy/sentinel-policies.schema";
import { trpc } from "@/lib/trpc/client";
import { zodResolver } from "@hookform/resolvers/zod";
import { ChevronDown, DoubleChevronRight } from "@unkey/icons";
import {
  Button,
  FormInput,
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
  SlidePanel,
} from "@unkey/ui";
import { FormLabel } from "@unkey/ui/src/components/form/form-helpers";
import { useState } from "react";
import { Controller, FormProvider, useForm, useWatch } from "react-hook-form";
import { AccordionSection } from "./accordion-section";
import {
  MatchConditionEditorBody,
  MatchConditionsCollapsedSection,
  MatchConditionsSummary,
} from "./match-condition-editor";
import { PolicyConfigFields } from "./policy-config-fields";
import { PolicySummary } from "./policy-summaries";
import {
  POLICY_TYPE_OPTIONS,
  type PolicyFormValues,
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
  onAdd: (prodPolicy: SentinelPolicy | null, previewPolicy: SentinelPolicy | null) => void;
};

type EditProps = CommonProps & {
  mode: "edit";
  initialPolicy: SentinelPolicy;
  onSave: (updated: SentinelPolicy) => void;
};

export type SentinelPolicyPanelProps = AddProps | EditProps;

export function SentinelPolicyPanel(props: SentinelPolicyPanelProps) {
  const { envASlug, envBSlug, isOpen, topOffset, onClose } = props;
  const isEdit = props.mode === "edit";

  const form = useForm<PolicyFormValues>({
    resolver: zodResolver(policyFormSchema),
    defaultValues: isEdit
      ? fromSentinelPolicy(props.initialPolicy, "__all__")
      : // For now we only have keyauth so we default to that
        getDefaultValues("keyauth"),
  });
  const { control, handleSubmit, reset } = form;

  // Only watch the discriminator at this level — summary subscriptions live in
  // their own components below so a keystroke in one form field doesn't
  // re-render the entire panel (form, accordion, both summaries, keyspace
  // query, condition editor).
  const watchedType = useWatch({ control, name: "type" });

  const { data: availableKeyspaces = {} } =
    trpc.deploy.environmentSettings.getAvailableKeyspaces.useQuery();
  const keyspaceNames: Record<string, string> = Object.fromEntries(
    Object.entries(availableKeyspaces).map(([id, ks]) => [id, ks?.api?.name ?? id]),
  );

  type ExpandedSection = "config" | "matchConditions" | "none";
  const [expanded, setExpanded] = useState<ExpandedSection>("config");
  const toggleSection = (section: Exclude<ExpandedSection, "none">) =>
    setExpanded((prev) => (prev === section ? "none" : section));

  const onSubmit = (values: PolicyFormValues) => {
    if (props.mode === "edit") {
      const updated = toSentinelPolicy(values, props.initialPolicy.id);
      props.onSave({ ...updated, enabled: props.initialPolicy.enabled });
      onClose();
      return;
    }

    const policy = toSentinelPolicy(values);
    const prodPolicy =
      values.environmentId === "__all__" || values.environmentId === envASlug
        ? { ...policy, enabled: true }
        : null;
    const previewPolicy =
      values.environmentId === "__all__" || values.environmentId === envBSlug
        ? { ...policy, enabled: true }
        : null;
    props.onAdd(prodPolicy, previewPolicy);
    onClose();
    reset(getDefaultValues("keyauth"));
  };

  return (
    <SlidePanel.Root isOpen={isOpen} onClose={onClose} topOffset={topOffset}>
      <SlidePanel.Header>
        <div className="flex flex-col">
          <span className="text-gray-12 font-medium text-base leading-8">
            {isEdit ? "Edit Policy" : "Add Policy"}
          </span>
          <span className="text-gray-11 text-[13px] leading-5">
            {isEdit ? "Update this sentinel policy." : "Configure and add a new sentinel policy."}
          </span>
        </div>
        <SlidePanel.Close
          aria-label="Close panel"
          className="mt-0.5 inline-flex items-center justify-center size-9 rounded-md hover:bg-grayA-3 transition-colors cursor-pointer"
        >
          <DoubleChevronRight
            iconSize="lg-medium"
            className="text-gray-10 transition-transform duration-300 ease-out group-hover:text-gray-12"
          />
        </SlidePanel.Close>
      </SlidePanel.Header>

      <SlidePanel.Content>
        <FormProvider {...form}>
          <form onSubmit={handleSubmit(onSubmit)} className="h-full flex flex-col">
            <div className="flex-1 overflow-y-auto pt-6 bg-grayA-2">
              <div className="flex flex-col gap-5 px-8">
                <Controller
                  control={control}
                  name="name"
                  render={({ field, fieldState }) => (
                    <FormInput
                      label="Name"
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
                    <fieldset className="flex flex-col gap-1.5 border-0 m-0 p-0">
                      <FormLabel
                        label="Type"
                        htmlFor="policy-type-select"
                        tooltipContent="The kind of protection this policy enforces."
                      />
                      <Select value={field.value} onValueChange={field.onChange} disabled={isEdit}>
                        <SelectTrigger
                          id="policy-type-select"
                          className="capitalize"
                          rightIcon={
                            <ChevronDown className="absolute right-2" iconSize="md-medium" />
                          }
                        >
                          <SelectValue />
                        </SelectTrigger>
                        <SelectContent>
                          {POLICY_TYPE_OPTIONS.map((opt) => (
                            <SelectItem key={opt.value} value={opt.value}>
                              {opt.label}
                            </SelectItem>
                          ))}
                        </SelectContent>
                      </Select>
                    </fieldset>
                  )}
                />
              </div>

              {expanded === "config" && (
                <div className="mt-6">
                  <AccordionSection
                    label="Policy Configuration"
                    summary={<PolicySummary keyspaceNames={keyspaceNames} />}
                    active
                    onToggle={() => toggleSection("config")}
                  >
                    <PolicyConfigFields type={watchedType} control={control} />
                  </AccordionSection>
                </div>
              )}
              {expanded === "matchConditions" && (
                <div className="mt-6">
                  <AccordionSection
                    label="Match Conditions"
                    summary={<MatchConditionsSummary />}
                    active
                    onToggle={() => toggleSection("matchConditions")}
                    tooltipContent={
                      <span>
                        All conditions must match (
                        <span className="text-gray-12 font-medium">AND</span> logic).
                      </span>
                    }
                  >
                    <MatchConditionEditorBody />
                  </AccordionSection>
                </div>
              )}
            </div>

            {expanded !== "config" && (
              <AccordionSection
                label="Policy Configuration"
                summary={<PolicySummary keyspaceNames={keyspaceNames} />}
                active={false}
                onToggle={() => toggleSection("config")}
              >
                {null}
              </AccordionSection>
            )}
            {expanded !== "matchConditions" && (
              <MatchConditionsCollapsedSection onToggle={() => toggleSection("matchConditions")} />
            )}

            <div className="border-t border-grayA-4">
              <div className="px-8 py-6">
                <Controller
                  control={control}
                  name="environmentId"
                  render={({ field }) => (
                    <fieldset className="flex flex-col gap-1.5 border-0 m-0 p-0">
                      <FormLabel
                        label="Environment"
                        htmlFor="policy-env-select"
                        tooltipContent="Which environments this policy will be added to."
                      />
                      <Select value={field.value} onValueChange={field.onChange} disabled={isEdit}>
                        <SelectTrigger
                          id="policy-env-select"
                          className="capitalize"
                          rightIcon={
                            <ChevronDown className="absolute right-2" iconSize="md-medium" />
                          }
                        >
                          <SelectValue />
                        </SelectTrigger>
                        <SelectContent>
                          <SelectItem value="__all__">All Environments</SelectItem>
                          <SelectItem value={envASlug} className="capitalize">
                            {envASlug}
                          </SelectItem>
                          <SelectItem value={envBSlug} className="capitalize">
                            {envBSlug}
                          </SelectItem>
                        </SelectContent>
                      </Select>
                    </fieldset>
                  )}
                />
              </div>
            </div>

            <div className="border-t border-gray-4 bg-white dark:bg-black px-8 py-5 flex items-center justify-end">
              <Button type="submit" variant="primary" size="md" className="px-3">
                {isEdit ? "Save Changes" : "Add Policy"}
              </Button>
            </div>
          </form>
        </FormProvider>
      </SlidePanel.Content>
    </SlidePanel.Root>
  );
}
