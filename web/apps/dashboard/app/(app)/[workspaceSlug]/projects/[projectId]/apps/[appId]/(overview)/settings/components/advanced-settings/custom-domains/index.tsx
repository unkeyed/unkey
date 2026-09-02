"use client";

import { zodResolver } from "@hookform/resolvers/zod";
import {
  AlertBanner,
  AlertBannerActions,
  AlertBannerDescription,
  AlertBannerTitle,
  Button,
  FormInput,
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@unkey/ui";
import Link from "next/link";
import { IconChevronDownOutline18, IconLink4Outline18 } from "nucleo-ui-outline-18";
import { useEffect, useState } from "react";
import { Controller, useForm } from "react-hook-form";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { collection } from "@/lib/collections";
import {
  type CustomDomain,
  isCustomDomainLimitError,
} from "@/lib/collections/deploy/custom-domains";
import { useBillingUIUpgrades } from "@/lib/flags/use-billing-ui-upgrades";
import { routes } from "@/lib/navigation/routes";
import { getErrorMessage } from "@/lib/unkey-client";
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
  const workspace = useWorkspaceNavigation();
  const [expanded, setExpanded] = useState(false);
  const [limitMessage, setLimitMessage] = useState<string | null>(null);
  useEffect(() => {
    if (window.location.hash.slice(1) === "custom-domains") {
      setExpanded(true);
    }
  }, []);

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

  const onSubmit = async (values: CustomDomainFormValues) => {
    const trimmedDomain = values.domain.trim();
    if (customDomains.some((d) => d.domain === trimmedDomain)) {
      setError("domain", { message: "Domain already registered" });
      return;
    }
    const appId = environments.find((e) => e.id === values.environmentId)?.appId ?? "";

    setLimitMessage(null);
    const tx = collection.customDomains.insert(
      {
        id: crypto.randomUUID(),
        domain: trimmedDomain,
        projectId,
        appId,
        environmentId: values.environmentId,
        verificationStatus: "pending",
        dnsRecords: [],
        verificationError: null,
        domainConnectProvider: null,
        domainConnectUrl: null,
        createdAt: Date.now(),
        updatedAt: null,
      },
      { metadata: { workspaceSlug: workspace.slug } },
    );

    try {
      await tx.isPersisted.promise;
      reset({ environmentId: values.environmentId, domain: "" });
    } catch (err) {
      if (isCustomDomainLimitError(err)) {
        setLimitMessage(getErrorMessage(err));
        return;
      }
      console.error("Failed to add custom domain", err);
    }
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
      icon={<IconLink4Outline18 className="text-gray-12" />}
      title="Custom Domains"
      description="Serve your deployment from your own domain name"
      displayValue={displayValue}
      onSubmit={handleSubmit(onSubmit)}
      saveState={saveState}
      expanded={expanded}
      onExpandedChange={setExpanded}
      stickyHeader={limitMessage ? <LimitBanner message={limitMessage} /> : undefined}
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
                  rightIcon={<IconChevronDownOutline18 className="absolute right-3 opacity-70" />}
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

const LimitBanner = ({ message }: { message: string }) => {
  const workspace = useWorkspaceNavigation();
  const billingUpgrades = useBillingUIUpgrades();

  return (
    <AlertBanner variant="error" className="mb-2">
      <AlertBannerTitle>Custom domain limit reached</AlertBannerTitle>
      <AlertBannerDescription>{message}</AlertBannerDescription>
      <AlertBannerActions>
        {billingUpgrades && (
          <Button
            variant="outline"
            size="sm"
            className="px-3"
            render={<Link href={routes.settings.limits({ workspaceSlug: workspace.slug })} />}
          >
            View limits
          </Button>
        )}
        <Button
          variant="primary"
          size="sm"
          className="px-3"
          render={
            <Link
              href={routes.settings.billing({
                workspaceSlug: workspace.slug,
                intent: "compute",
              })}
            />
          }
        >
          Upgrade plan
        </Button>
      </AlertBannerActions>
    </AlertBanner>
  );
};
