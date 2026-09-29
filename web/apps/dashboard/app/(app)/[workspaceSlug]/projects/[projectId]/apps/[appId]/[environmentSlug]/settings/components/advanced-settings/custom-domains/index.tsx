"use client";

import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { collection } from "@/lib/collections";
import {
  type CustomDomain,
  isCustomDomainLimitError,
} from "@/lib/collections/deploy/custom-domains";
import { useBillingUIUpgrades } from "@/lib/flags/use-billing-ui-upgrades";
import { routes } from "@/lib/navigation/routes";
import { getErrorMessage } from "@/lib/unkey-client";
import { zodResolver } from "@hookform/resolvers/zod";
import { IconLink4Outline18 } from "@unkey/icons";
import {
  AlertBanner,
  AlertBannerActions,
  AlertBannerDescription,
  AlertBannerTitle,
  Button,
  FormInput,
} from "@unkey/ui";
import Link from "next/link";
import { useEffect, useState } from "react";
import { useForm } from "react-hook-form";
import { useAppId, useProjectData } from "../../../../../data-provider";
import { useEnvironmentSettings } from "../../../environment-provider";
import { SettingField, WideContent } from "../../shared/form-blocks";
import { FormSettingCard, resolveSaveState } from "../../shared/form-setting-card";
import { CustomDomainRow } from "./custom-domain-row";
import { type CustomDomainFormValues, customDomainSchema } from "./schema";

export const CustomDomains = () => {
  const { customDomains, projectId } = useProjectData();
  const appId = useAppId();
  const {
    settings: { environmentId },
  } = useEnvironmentSettings();

  return (
    <CustomDomainSettings
      customDomains={customDomains.filter((d) => d.environmentId === environmentId)}
      projectId={projectId}
      appId={appId}
      environmentId={environmentId}
    />
  );
};

type CustomDomainSettingsProps = {
  customDomains: CustomDomain[];
  projectId: string;
  appId: string;
  environmentId: string;
};

const CustomDomainSettings: React.FC<CustomDomainSettingsProps> = ({
  customDomains,
  projectId,
  appId,
  environmentId,
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
    register,
    reset,
    setError,
    formState: { isValid, isSubmitting, errors },
  } = useForm<CustomDomainFormValues>({
    resolver: zodResolver(customDomainSchema),
    mode: "onChange",
    defaultValues: { domain: "" },
  });

  const onSubmit = async (values: CustomDomainFormValues) => {
    const trimmedDomain = values.domain.trim();
    if (customDomains.some((d) => d.domain === trimmedDomain)) {
      setError("domain", { message: "Domain already registered" });
      return;
    }
    setLimitMessage(null);
    const tx = collection.customDomains.insert(
      {
        id: crypto.randomUUID(),
        domain: trimmedDomain,
        projectId,
        appId,
        environmentId,
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
      reset({ domain: "" });
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
        <FormInput
          label="Domain"
          placeholder="api.example.com"
          className="[&_input]:font-mono"
          error={errors.domain?.message}
          {...register("domain")}
        />
      </SettingField>
      <WideContent>
        {customDomains.length > 0 && (
          <div className="border rounded-lg overflow-hidden mt-1 bg-raised">
            {customDomains.map((d) => (
              <CustomDomainRow key={d.id} domain={d} />
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
