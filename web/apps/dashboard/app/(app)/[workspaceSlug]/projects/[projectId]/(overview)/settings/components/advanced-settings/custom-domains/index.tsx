"use client";

import { collection } from "@/lib/collections";
import type { CustomDomain } from "@/lib/collections/deploy/custom-domains";
import { zodResolver } from "@hookform/resolvers/zod";
import { ChevronDown, Link4 } from "@unkey/icons";
import {
  Checkbox,
  FormInput,
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@unkey/ui";
import { Controller, useForm, useWatch } from "react-hook-form";
import { useProjectData } from "../../../../data-provider";
import { useEnvironmentSettings } from "../../../environment-provider";
import { SettingField, WideContent } from "../../shared/form-blocks";
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
      alsoAddWww: true,
    },
  });

  // Watch the domain so we can default the "also add www" checkbox sensibly:
  // checked for apex-looking inputs (no leading "www.", at least two labels),
  // hidden otherwise. The checkbox stays in the form state regardless so the
  // submit handler can read it without a separate ref.
  const domainValue = useWatch({ control, name: "domain" });
  const wwwSuggestionEligible =
    typeof domainValue === "string" &&
    domainValue.trim().length > 0 &&
    !/^www\./i.test(domainValue.trim()) &&
    domainValue.trim().split(".").length >= 2;

  // Clones the same insert payload twice; the collection layer enforces
  // server-side uniqueness so the second insert fails cleanly if the user
  // already owns www.<domain>.
  const insertDomain = (domain: string, environmentId: string, appId: string) => {
    collection.customDomains.insert({
      id: crypto.randomUUID(),
      domain,
      workspaceId: "",
      projectId,
      appId,
      environmentId,
      verificationStatus: "pending",
      verificationToken: "",
      ownershipVerified: false,
      cnameVerified: false,
      targetCname: "",
      checkAttempts: 0,
      lastCheckedAt: null,
      verificationError: null,
      domainConnectProvider: null,
      domainConnectUrl: null,
      createdAt: Date.now(),
      updatedAt: null,
    });
  };

  const onSubmit = (values: CustomDomainFormValues) => {
    const trimmedDomain = values.domain.trim();
    if (customDomains.some((d) => d.domain === trimmedDomain)) {
      setError("domain", { message: "Domain already registered" });
      return;
    }
    const appId = environments.find((e) => e.id === values.environmentId)?.appId ?? "";

    insertDomain(trimmedDomain, values.environmentId, appId);

    // Companion www domain. The dashboard does not pre-populate the
    // StripWWW redirect on the new route — that lives on a frontline_route
    // which only exists after verification. The user flips the toggle on
    // the www row's redirect panel once it's verified.
    if (
      values.alsoAddWww &&
      wwwSuggestionEligible &&
      !customDomains.some((d) => d.domain === `www.${trimmedDomain}`)
    ) {
      insertDomain(`www.${trimmedDomain}`, values.environmentId, appId);
    }

    reset({ environmentId: values.environmentId, domain: "", alsoAddWww: true });
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
        {wwwSuggestionEligible && (
          <div className="flex items-center gap-2 pl-[152px]">
            <Controller
              control={control}
              name="alsoAddWww"
              render={({ field }) => (
                <Checkbox
                  id="alsoAddWww"
                  checked={field.value}
                  onCheckedChange={(checked) => field.onChange(checked === true)}
                />
              )}
            />
            <label htmlFor="alsoAddWww" className="text-[13px] text-gray-11 cursor-pointer">
              Also add <span className="font-mono text-gray-12">www.{domainValue.trim()}</span>{" "}
              <span className="text-gray-9">(recommended)</span>
            </label>
          </div>
        )}
      </SettingField>
      <WideContent>
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
      </WideContent>
    </FormSettingCard>
  );
};
