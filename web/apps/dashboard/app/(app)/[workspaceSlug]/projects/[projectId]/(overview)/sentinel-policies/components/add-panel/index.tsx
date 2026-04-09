"use client";

import type { SentinelPolicy } from "@/lib/collections/deploy/sentinel-policies.schema";
import { trpc } from "@/lib/trpc/client";
import { zodResolver } from "@hookform/resolvers/zod";
import { DoubleChevronRight } from "@unkey/icons";
import { Button, FormInput, FormSelect, SlidePanel } from "@unkey/ui";
import { useCallback, useState } from "react";
import { Controller, type FieldErrors, FormProvider, useForm, useWatch } from "react-hook-form";
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

// Fields always visible or with their own accordion section — everything else is "config".
const NON_CONFIG_KEYS = new Set(["name", "type", "environmentId", "matchConditions"]);

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

  const onInvalid = useCallback((fieldErrors: FieldErrors<PolicyFormValues>) => {
    const hasConfigError = Object.keys(fieldErrors).some((k) => !NON_CONFIG_KEYS.has(k));
    const hasMatchError = Boolean(fieldErrors.matchConditions);

    // Expand the first section that has errors so the user sees them.
    const target: ExpandedSection | null = hasConfigError
      ? "config"
      : hasMatchError
        ? "matchConditions"
        : null;

    if (target) {
      setExpanded(target);
    }

    // After React renders the expanded section, focus the first errored element.
    requestAnimationFrame(() => {
      const el = document.querySelector(
        '[aria-invalid="true"], [data-error="true"]',
      ) as HTMLElement | null;
      el?.focus();
      el?.scrollIntoView({ behavior: "smooth", block: "center" });
    });
  }, []);

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

    if (props.mode === "edit") {
      props.onSave(prodPolicy, previewPolicy);
      onClose();
      return;
    }

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
          <form onSubmit={handleSubmit(onSubmit, onInvalid)} className="h-full flex flex-col">
            <div className="flex-1 overflow-y-auto pt-6 bg-grayA-2">
              <div className="flex flex-col gap-5 px-8">
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
                      onValueChange={field.onChange}
                      disabled={isEdit}
                      description="The kind of protection this policy enforces."
                      descriptionPosition="label"
                      triggerClassName="capitalize"
                    />
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
                    <PolicyConfigFields type={watchedType} />
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
          </form>
        </FormProvider>
      </SlidePanel.Content>
    </SlidePanel.Root>
  );
}
