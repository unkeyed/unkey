"use client";

import type { SentinelPolicy } from "@/lib/trpc/routers/deploy/environment-settings/sentinel/update-middleware";
import { zodResolver } from "@hookform/resolvers/zod";
import { match } from "@unkey/match";
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
import { FormDescription } from "@unkey/ui/src/components/form/form-helpers";
import { type Control, Controller, useForm } from "react-hook-form";
import { KeyAuthFields } from "./forms/keyauth-fields";
import { RateLimitFields } from "./forms/ratelimit-fields";
import { MatchConditionEditor } from "./match-condition-editor";
import {
  POLICY_TYPE_OPTIONS,
  type PolicyFormValues,
  type PolicyType,
  getDefaultValues,
  policyFormSchema,
  toSentinelPolicy,
} from "./schema";

type SentinelPolicyAddPanelProps = {
  envASlug: string;
  envBSlug: string;
  isOpen: boolean;
  topOffset: number;
  onClose: () => void;
  onAdd: (prodPolicy: SentinelPolicy | null, previewPolicy: SentinelPolicy | null) => void;
};

function PolicyConfigFields({ type, control }: { type: PolicyType; control: Control<PolicyFormValues> }) {
  return match(type)
    .with("keyauth", () => (
      <KeyAuthFields control={control as Control<Extract<PolicyFormValues, { type: "keyauth" }>>} />
    ))
    .with("ratelimit", () => (
      <RateLimitFields
        control={control as Control<Extract<PolicyFormValues, { type: "ratelimit" }>>}
      />
    ))
    .exhaustive();
}

export function SentinelPolicyAddPanel({
  envASlug,
  envBSlug,
  isOpen,
  topOffset,
  onClose,
  onAdd,
}: SentinelPolicyAddPanelProps) {
  const {
    control,
    handleSubmit,
    watch,
    reset,
  } = useForm<PolicyFormValues>({
    resolver: zodResolver(policyFormSchema),
    defaultValues: getDefaultValues("ratelimit"),
  });

  const watchedType = watch("type");

  const handleTypeChange = (newType: PolicyType) => {
    const currentValues = watch();
    reset({
      ...getDefaultValues(newType),
      name: currentValues.name,
      environmentId: currentValues.environmentId,
      matchConditions: currentValues.matchConditions,
    });
  };

  const onSubmit = (values: PolicyFormValues) => {
    const policy = toSentinelPolicy(values);

    const prodPolicy =
      values.environmentId === "__all__" || values.environmentId === envASlug ? policy : null;
    const previewPolicy =
      values.environmentId === "__all__" || values.environmentId === envBSlug
        ? { ...policy, enabled: values.environmentId === envBSlug }
        : null;

    onAdd(prodPolicy, previewPolicy);
    onClose();
    reset(getDefaultValues("ratelimit"));
  };

  return (
    <SlidePanel.Root isOpen={isOpen} onClose={onClose} topOffset={topOffset}>
      <SlidePanel.Header>
        <div className="flex flex-col">
          <span className="text-gray-12 font-medium text-base leading-8">Add Policy</span>
          <span className="text-gray-11 text-[13px] leading-5">
            Configure and add a new sentinel policy.
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
        <form onSubmit={handleSubmit(onSubmit)} className="h-full flex flex-col">
          <div className="flex-1 overflow-y-auto pt-6 bg-grayA-2">
            <div className="flex flex-col gap-5 px-8">
              <Controller
                control={control}
                name="name"
                render={({ field, fieldState }) => (
                  <FormInput
                    label="Name"
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
                    <label htmlFor="policy-type-select" className="text-gray-11 text-[13px]">
                      Type
                    </label>
                    <Select
                      value={field.value}
                      onValueChange={(v) => handleTypeChange(v as PolicyType)}
                    >
                      <SelectTrigger
                        id="policy-type-select"
                        className="capitalize"
                        aria-describedby="policy-type-desc"
                        rightIcon={
                          <ChevronDown className="absolute right-2" iconSize="md-medium" />
                        }
                      >
                        <SelectValue />
                      </SelectTrigger>
                      <SelectContent className="z-60">
                        {POLICY_TYPE_OPTIONS.map((opt) => (
                          <SelectItem key={opt.value} value={opt.value}>
                            {opt.label}
                          </SelectItem>
                        ))}
                      </SelectContent>
                    </Select>
                    <FormDescription
                      description="The kind of protection this policy enforces."
                      descriptionId="policy-type-desc"
                      errorId="policy-type-error"
                    />
                  </fieldset>
                )}
              />

              <PolicyConfigFields type={watchedType} control={control} />
            </div>
          </div>

          <Controller
            control={control}
            name="matchConditions"
            render={({ field }) => (
              <MatchConditionEditor conditions={field.value} onChange={field.onChange} />
            )}
          />

          <div className="border-t border-grayA-4">
            <div className="px-8 py-6">
              <Controller
                control={control}
                name="environmentId"
                render={({ field }) => (
                  <fieldset className="flex flex-col gap-1.5 border-0 m-0 p-0">
                    <label htmlFor="policy-env-select" className="text-gray-11 text-[13px]">
                      Environment
                    </label>
                    <Select value={field.value} onValueChange={field.onChange}>
                      <SelectTrigger
                        id="policy-env-select"
                        className="capitalize"
                        aria-describedby="policy-env-desc"
                        rightIcon={
                          <ChevronDown className="absolute right-2" iconSize="md-medium" />
                        }
                      >
                        <SelectValue />
                      </SelectTrigger>
                      <SelectContent className="z-60">
                        <SelectItem value="__all__">All Environments</SelectItem>
                        <SelectItem value={envASlug} className="capitalize">
                          {envASlug}
                        </SelectItem>
                        <SelectItem value={envBSlug} className="capitalize">
                          {envBSlug}
                        </SelectItem>
                      </SelectContent>
                    </Select>
                    <FormDescription
                      description="Which environment(s) this policy will be added to."
                      descriptionId="policy-env-desc"
                      errorId="policy-env-error"
                    />
                  </fieldset>
                )}
              />
            </div>
          </div>

          <div className="border-t border-gray-4 bg-white dark:bg-black px-8 py-5 flex items-center justify-end">
            <Button type="submit" variant="primary" size="md" className="px-3">
              Add Policy
            </Button>
          </div>
        </form>
      </SlidePanel.Content>
    </SlidePanel.Root>
  );
}
