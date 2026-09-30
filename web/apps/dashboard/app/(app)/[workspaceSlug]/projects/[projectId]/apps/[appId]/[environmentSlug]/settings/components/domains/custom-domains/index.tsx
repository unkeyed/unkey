"use client";

import { PlansScreen } from "@/app/(app)/[workspaceSlug]/settings/billing/components/plans-screen";
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
import { IconEarthOutline18 } from "@unkey/icons";
import {
  AlertBanner,
  AlertBannerActions,
  AlertBannerDescription,
  AlertBannerTitle,
  Button,
  EmptyState,
  EmptyStateDescription,
  EmptyStateHeader,
  EmptyStateIcon,
  EmptyStateTitle,
  FormInput,
} from "@unkey/ui";
import Link from "next/link";
import { useState } from "react";
import { useForm } from "react-hook-form";
import { useAppId, useProjectData } from "../../../../../data-provider";
import { useAppEnvironment } from "../../../../environment-context";
import { SettingsSection } from "../../shared/settings-section";
import { CustomDomainRow } from "./custom-domain-row";
import { type CustomDomainFormValues, customDomainSchema } from "./schema";

export function CustomDomains() {
  const { customDomains, projectId } = useProjectData();
  const appId = useAppId();
  const { environment } = useAppEnvironment();

  return (
    <CustomDomainSettings
      customDomains={customDomains.filter((d) => d.environmentId === environment.id)}
      projectId={projectId}
      appId={appId}
      environmentId={environment.id}
    />
  );
}

type CustomDomainSettingsProps = {
  customDomains: CustomDomain[];
  projectId: string;
  appId: string;
  environmentId: string;
};

function CustomDomainSettings({
  customDomains,
  projectId,
  appId,
  environmentId,
}: CustomDomainSettingsProps) {
  const workspace = useWorkspaceNavigation();
  const [limitMessage, setLimitMessage] = useState<string | null>(null);

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

  return (
    <SettingsSection title="Custom domains">
      {limitMessage ? <LimitBanner message={limitMessage} /> : null}
      <div className="overflow-hidden rounded-lg border bg-raised">
        {customDomains.length > 0 ? (
          customDomains.map((d) => <CustomDomainRow key={d.id} domain={d} />)
        ) : (
          <EmptyState frame="none">
            <EmptyStateIcon>
              <IconEarthOutline18 />
            </EmptyStateIcon>
            <EmptyStateHeader>
              <EmptyStateTitle>No custom domains</EmptyStateTitle>
              <EmptyStateDescription>
                Add a domain below to serve this app from your own hostname.
              </EmptyStateDescription>
            </EmptyStateHeader>
          </EmptyState>
        )}
        <form
          onSubmit={handleSubmit(onSubmit)}
          className="flex items-start gap-2 border-t bg-grayA-2 px-5 py-3"
        >
          <FormInput
            aria-label="Domain"
            placeholder="api.example.com"
            className="flex-1 [&_input]:font-mono"
            error={errors.domain?.message}
            {...register("domain")}
          />
          <Button
            type="submit"
            variant="primary"
            size="lg"
            className="px-3"
            disabled={!isValid}
            loading={isSubmitting}
          >
            Add domain
          </Button>
        </form>
      </div>
    </SettingsSection>
  );
}

const LimitBanner = ({ message }: { message: string }) => {
  const workspace = useWorkspaceNavigation();
  const billingUpgrades = useBillingUIUpgrades();
  const [plansOpen, setPlansOpen] = useState(false);

  return (
    <AlertBanner variant="error">
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
        <Button variant="primary" size="sm" className="px-3" onClick={() => setPlansOpen(true)}>
          Upgrade plan
        </Button>
      </AlertBannerActions>
      <PlansScreen open={plansOpen} onOpenChange={setPlansOpen} reason="custom-domains" />
    </AlertBanner>
  );
};
