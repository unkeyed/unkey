"use client";

import { collection } from "@/lib/collections";
import type { CustomDomain } from "@/lib/collections/deploy/custom-domains";
import { zodResolver } from "@hookform/resolvers/zod";
import { ChevronDown, Link4 } from "@unkey/icons";
import {
  FormInput,
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@unkey/ui";
import { Controller, useForm } from "react-hook-form";
import { useProjectData } from "../../../../data-provider";
import { useEnvironmentSettings } from "../../../environment-provider";
import { SettingField } from "../../shared/form-blocks";
import { FormSettingCard, resolveSaveState } from "../../shared/form-setting-card";
import { CustomDomainRow } from "./custom-domain-row";
import { type CustomDomainFormValues, customDomainSchema } from "./schema";

export const CustomDomains = () => {
  const { environments, customDomains, projectId } = useProjectData();
  const {
    settings: { environmentId: defaultEnvironmentId },
  } = useEnvironmentSettings();

  return (
    <CustomDomainSettings
      environments={environments}
      customDomains={customDomains}
      projectId={projectId}
      defaultEnvironmentId={defaultEnvironmentId}
    />
  );
};

type CustomDomainSettingsProps = {
  environments: { id: string; slug: string; appId: string }[];
  customDomains: CustomDomain[];
  projectId: string;
  defaultEnvironmentId: string;
};

const CustomDomainSettings: React.FC<CustomDomainSettingsProps> = ({
  environments,
  customDomains,
  projectId,
  defaultEnvironmentId,
}) => {
  const {
    handleSubmit,
    control,
    register,
    reset,
    setError,
    formState: { isValid, isSubmitting, errors },
  } = useForm<CustomDomainFormValues>({
    resolver: zodResolver(customDomainSchema),
    mode: "onChange",
    defaultValues: {
      environmentId: defaultEnvironmentId,
      domain: "",
    },
  });

  const onSubmit = (values: CustomDomainFormValues) => {
    const trimmedDomain = values.domain.trim();
    if (customDomains.some((d) => d.domain === trimmedDomain)) {
      setError("domain", { message: "Domain already registered" });
      return;
    }
    const appId = environments.find((e) => e.id === values.environmentId)?.appId ?? "";

    collection.customDomains.insert({
      id: crypto.randomUUID(),
      domain: trimmedDomain,
      workspaceId: "",
      projectId,
      appId,
      environmentId: values.environmentId,
      verificationStatus: "pending",
      verificationToken: "",
      ownershipVerified: false,
      cnameVerified: false,
      targetCname: "",
      checkAttempts: 0,
      lastCheckedAt: null,
      verificationError: null,
      createdAt: Date.now(),
      updatedAt: null,
    });
    reset({ environmentId: values.environmentId, domain: "" });
  };

  const saveState = resolveSaveState([
    [isSubmitting, { status: "saving" }],
    [!isValid, { status: "disabled" }],
  ]);

  const displayValue =
    customDomains.length === 0 ? null : (
      <div className="space-x-1">
        <span className="font-medium text-gray-12">{customDomains.length}</span>
        <span className="text-gray-11 font-normal">
          domain{customDomains.length !== 1 ? "s" : ""}
        </span>
      </div>
    );

  return (
    <FormSettingCard
      icon={<Link4 className="text-gray-12" iconSize="xl-medium" />}
      title="Custom Domains"
      description="Serve your deployment from your own domain name"
      displayValue={displayValue}
      onSubmit={handleSubmit(onSubmit)}
      saveState={saveState}
    >
      <SettingField>
        <div className="flex items-center gap-3">
          <span className="text-[13px] text-gray-11 w-35">Environment</span>
          <span className="flex-1 text-[13px] text-gray-11">Domain</span>
        </div>
        <div className="flex items-start gap-3">
          <Controller
            control={control}
            name="environmentId"
            render={({ field }) => (
              <Select value={field.value} onValueChange={field.onChange}>
                <SelectTrigger
                  wrapperClassName="w-[140px]"
                  variant={errors.environmentId ? "error" : "default"}
                  rightIcon={<ChevronDown className="absolute right-3 size-3 opacity-70" />}
                >
                  <SelectValue placeholder="Environment">
                    {environments.find((e) => e.id === field.value)?.slug ?? ""}
                  </SelectValue>
                </SelectTrigger>
                <SelectContent>
                  {environments.map((env) => (
                    <SelectItem key={env.id} value={env.id}>
                      {env.slug}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            )}
          />
          <FormInput
            placeholder="api.example.com"
            className="flex-1 [&_input]:font-mono"
            error={errors.domain?.message}
            {...register("domain")}
          />
        </div>

        {customDomains.length > 0 && (
          <div className="border border-gray-4 rounded-lg overflow-hidden mt-1 dark:bg-black bg-white">
            {customDomains.map((d) => (
              <CustomDomainRow
                key={d.id}
                domain={d}
                environmentSlug={environments.find((e) => e.id === d.environmentId)?.slug}
              />
            ))}
          </div>
        )}
      </SettingField>
    </FormSettingCard>
  );
};
