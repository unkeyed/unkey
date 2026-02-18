"use client";

import { collection } from "@/lib/collections";
import { zodResolver } from "@hookform/resolvers/zod";
import { ChevronDown, Gear, Trash } from "@unkey/icons";
import {
  Button,
  FormInput,
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@unkey/ui";
import { Controller, useForm } from "react-hook-form";
import { useProjectData } from "../../../../data-provider";
import { FormSettingCard } from "../../shared/form-setting-card";
import { type CustomDomainFormValues, customDomainSchema } from "./schema";

export const CustomDomains = () => {
  const { environments, customDomains, projectId } = useProjectData();

  const defaultEnvironmentId =
    environments.find((e) => e.slug === "production")?.id ?? environments[0]?.id ?? "";

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
  environments: Array<{ id: string; slug: string }>;
  customDomains: Array<{ id: string; domain: string; environmentId: string }>;
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
    collection.customDomains.insert({
      id: crypto.randomUUID(),
      domain: trimmedDomain,
      workspaceId: "",
      projectId,
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

  const displayValue = () => {
    if (customDomains.length === 0) {
      return <span className="text-gray-11">No custom domains</span>;
    }
    return (
      <div className="space-x-1">
        <span className="font-medium text-gray-12">{customDomains.length}</span>
        <span className="text-gray-11 font-normal">
          domain{customDomains.length !== 1 ? "s" : ""}
        </span>
      </div>
    );
  };

  return (
    <FormSettingCard
      icon={<Gear className="text-gray-12" iconSize="xl-medium" />}
      title="Custom Domains"
      description="Serve your deployment from your own domain name"
      displayValue={displayValue()}
      onSubmit={handleSubmit(onSubmit)}
      canSave={isValid && !isSubmitting}
      isSaving={isSubmitting}
    >
      <div className="flex flex-col gap-3 w-[480px]">
        <div className="flex items-center gap-3">
          <span className="text-[13px] text-gray-11 w-[140px]">Environment</span>
          <span className="flex-1 text-[13px] text-gray-11">Domain</span>
        </div>
        <div className="flex items-start gap-3">
          <Controller
            control={control}
            name="environmentId"
            render={({ field }) => (
              <Select value={field.value} onValueChange={field.onChange}>
                <SelectTrigger
                  className="h-9"
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
            className="flex-1 [&_input]:h-9 [&_input]:font-mono"
            error={errors.domain?.message}
            {...register("domain")}
          />
        </div>

        {customDomains.length > 0 && (
          <div className="flex flex-col gap-1 mt-1">
            {customDomains.map((d) => {
              const envSlug = environments.find((e) => e.id === d.environmentId)?.slug;
              return (
                <div key={d.id} className="flex items-center gap-2 py-1">
                  <span className="flex-1 text-[13px] font-mono text-gray-12">{d.domain}</span>
                  {envSlug && (
                    <span className="text-[11px] text-gray-11 bg-gray-3 px-1.5 py-0.5 rounded font-mono">
                      {envSlug}
                    </span>
                  )}
                  <Button
                    type="button"
                    variant="ghost"
                    size="sm"
                    className="px-1.5 text-error-11 hover:text-error-11"
                    onClick={() => collection.customDomains.delete(d.id)}
                  >
                    <Trash iconSize="sm-regular" />
                  </Button>
                </div>
              );
            })}
          </div>
        )}
      </div>
    </FormSettingCard>
  );
};
