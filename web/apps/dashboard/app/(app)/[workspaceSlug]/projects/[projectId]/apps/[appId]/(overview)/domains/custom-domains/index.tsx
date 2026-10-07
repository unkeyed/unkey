"use client";

import { PlansScreen } from "@/app/(app)/[workspaceSlug]/settings/billing/components/plans-screen";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { collection } from "@/lib/collections";
import { useCollectionLoad } from "@/lib/collections/use-collection-load";
import { useBillingUIUpgrades } from "@/lib/flags/use-billing-ui-upgrades";
import { routes } from "@/lib/navigation/routes";
import { IconEarthOutline18, IconPlusOutline18 } from "@unkey/icons";
import {
  AlertBanner,
  AlertBannerActions,
  AlertBannerDescription,
  AlertBannerTitle,
  Button,
  EmptyState,
  EmptyStateActions,
  EmptyStateDescription,
  EmptyStateHeader,
  EmptyStateIcon,
  EmptyStateTitle,
  PageBody,
  PageContainer,
  PageHeader,
  PageHeaderActions,
  PageHeaderContent,
  PageHeaderTitle,
  Skeleton,
} from "@unkey/ui";
import { cn } from "cn";
import Link from "next/link";
import { useState } from "react";
import { LoadError } from "../../../components/load-error";
import { useProjectData } from "../../data-provider";
import { platformDomainList } from "../model";
import { AddDomainDialog } from "./add-domain-dialog";
import { CUSTOM_DOMAIN_COLUMNS, CustomDomainRow } from "./custom-domain-row";
import { PlatformDomainRow } from "./platform-domain-row";

export function CustomDomainsPage() {
  const { environments, domains, customDomains, isDomainsLoading, isCustomDomainsLoading } =
    useProjectData();
  const domainsLoad = useCollectionLoad(collection.domains.utils, collection.customDomains.utils);
  const platformDomains = platformDomainList(domains, customDomains);
  const [addOpen, setAddOpen] = useState(false);
  const [limitMessage, setLimitMessage] = useState<string | null>(null);
  const [addedDomain, setAddedDomain] = useState<string | null>(null);
  const production = environments.find((e) => e.kind === "production") ?? environments.at(0);
  const openAdd = () => {
    setLimitMessage(null);
    setAddOpen(true);
  };
  const addButton = (
    <Button variant="primary" size="md" disabled={!production} onClick={openAdd}>
      <IconPlusOutline18 />
      Add domain
    </Button>
  );

  return (
    <PageContainer>
      <PageHeader>
        <PageHeaderContent>
          <PageHeaderTitle>Domains</PageHeaderTitle>
        </PageHeaderContent>
        <PageHeaderActions>{addButton}</PageHeaderActions>
      </PageHeader>
      <PageBody>
        {limitMessage ? <LimitBanner message={limitMessage} /> : null}
        <div className="overflow-hidden rounded-lg border bg-raised">
          {isCustomDomainsLoading || isDomainsLoading ? (
            <CustomDomainsSkeleton />
          ) : domainsLoad.failed ? (
            <LoadError title="Could not load domains" onRetry={domainsLoad.retry} />
          ) : customDomains.length + platformDomains.length > 0 ? (
            <>
              <CustomDomainsHeader />
              <div className="divide-y divide-grayA-4">
                {customDomains.map((d) => (
                  <CustomDomainRow
                    key={d.domain}
                    domain={d}
                    defaultExpanded={d.domain === addedDomain}
                    environment={environments.find((e) => e.id === d.environmentId)}
                  />
                ))}
                {platformDomains.map((d) => (
                  <PlatformDomainRow
                    key={d.id}
                    domain={d}
                    environment={environments.find((e) => e.id === d.environmentId)}
                  />
                ))}
              </div>
            </>
          ) : (
            <EmptyState frame="none">
              <EmptyStateIcon>
                <IconEarthOutline18 />
              </EmptyStateIcon>
              <EmptyStateHeader>
                <EmptyStateTitle>No custom domains yet</EmptyStateTitle>
                <EmptyStateDescription>
                  Serve this app from your own domain, such as api.example.com.
                </EmptyStateDescription>
              </EmptyStateHeader>
              <EmptyStateActions>{addButton}</EmptyStateActions>
            </EmptyState>
          )}
        </div>
        {production ? (
          <AddDomainDialog
            isOpen={addOpen}
            onOpenChange={setAddOpen}
            onLimitReached={setLimitMessage}
            onAdded={setAddedDomain}
            environments={environments}
            defaultEnvironmentId={production.id}
          />
        ) : null}
      </PageBody>
    </PageContainer>
  );
}

function CustomDomainsHeader() {
  return (
    <div
      className={cn(
        CUSTOM_DOMAIN_COLUMNS,
        "h-9 border-b bg-grayA-2 px-4 text-xs font-medium text-gray-9",
      )}
    >
      <span />
      <span>Domain</span>
      <span>Status</span>
      <span>Environment</span>
    </div>
  );
}

const SKELETON_ROWS = ["a", "b", "c"];

function CustomDomainsSkeleton() {
  return (
    <div aria-busy="true">
      <CustomDomainsHeader />
      <div aria-hidden="true" className="divide-y divide-grayA-4">
        {SKELETON_ROWS.map((key) => (
          <div key={key} className={cn(CUSTOM_DOMAIN_COLUMNS, "h-12 px-4")}>
            <Skeleton className="size-4 justify-self-center rounded-full" />
            <Skeleton className="h-3 w-48 max-w-full" />
            <Skeleton className="h-3 w-24 max-w-full" />
            <Skeleton className="h-3 w-20 max-w-full" />
          </div>
        ))}
      </div>
    </div>
  );
}

function LimitBanner({ message }: { message: string }) {
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
}
