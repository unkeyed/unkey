"use client";

import { PlansScreen } from "@/app/(app)/[workspaceSlug]/settings/billing/components/plans-screen";
import { LoadError } from "@/components/load-error";
import { ResourceSearchInput } from "@/components/resource-search-input";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { useBillingUIUpgrades } from "@/lib/flags/use-billing-ui-upgrades";
import { routes } from "@/lib/navigation/routes";
import { IconEarthOutline18, IconMagnifierOutline18, IconPlusOutline18 } from "@unkey/icons";
import { match } from "@unkey/match";
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
  ResourceList,
  ResourceListBody,
  ResourceListContent,
  ResourceListFooter,
  ResourceListHeader,
  ResourceListItem,
  Skeleton,
} from "@unkey/ui";
import Link from "next/link";
import { useState } from "react";
import { AddDomainDialog } from "./add-domain-dialog";
import { DomainColumns, DomainRow } from "./domain-row";
import { useDomainsPage } from "./use-domains-page";

export function CustomDomainsPage() {
  const {
    view,
    environments,
    searchKey,
    expanded,
    toggle,
    loadMore,
    clearSearch,
    retry,
    limitMessage,
    add,
  } = useDomainsPage();
  const addButton = (
    <Button variant="primary" size="md" disabled={!add.defaultEnvironment} onClick={add.openAdd}>
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
        <ResourceList>
          {view.state === "empty" ? null : (
            <ResourceListHeader>
              <ResourceSearchInput
                queryKey={searchKey}
                label="Search domains"
                placeholder="Search by domain or environment"
              />
            </ResourceListHeader>
          )}
          <ResourceListContent className="[--domain-columns:24px_minmax(0,1.6fr)_minmax(0,1fr)_minmax(0,0.7fr)_64px_12px]">
            {match(view)
              .with({ state: "loading" }, () => <DomainsSkeleton />)
              .with({ state: "failed" }, () => (
                <LoadError title="Could not load domains" onRetry={retry} />
              ))
              .with({ state: "no-match" }, ({ search: query }) => (
                <EmptyState frame="none">
                  <EmptyStateIcon>
                    <IconMagnifierOutline18 />
                  </EmptyStateIcon>
                  <EmptyStateHeader>
                    <EmptyStateTitle>No domains match "{query}"</EmptyStateTitle>
                    <EmptyStateDescription>
                      Search by another domain name or environment.
                    </EmptyStateDescription>
                  </EmptyStateHeader>
                  <EmptyStateActions>
                    <Button size="md" variant="outline" onClick={clearSearch}>
                      Clear search
                    </Button>
                  </EmptyStateActions>
                </EmptyState>
              ))
              .with({ state: "list" }, ({ domains: list, hasMore }) => (
                <>
                  <DomainsHeader />
                  <ResourceListBody>
                    {list.map((d) => (
                      <DomainRow
                        key={d.hostname}
                        domain={d}
                        expanded={expanded.has(d.hostname)}
                        onToggle={() => toggle(d.hostname)}
                        environment={environments.find((e) => e.id === d.environmentId)}
                      />
                    ))}
                  </ResourceListBody>
                  {hasMore ? (
                    <ResourceListFooter>
                      <Button size="md" variant="outline" onClick={loadMore}>
                        Load more
                      </Button>
                    </ResourceListFooter>
                  ) : null}
                </>
              ))
              .with({ state: "empty" }, () => (
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
              ))
              .exhaustive()}
          </ResourceListContent>
        </ResourceList>
        {add.defaultEnvironment ? (
          <AddDomainDialog
            isOpen={add.open}
            onSettled={add.onSettled}
            environments={environments}
            defaultEnvironmentId={add.defaultEnvironment.id}
          />
        ) : null}
      </PageBody>
    </PageContainer>
  );
}

function DomainsHeader() {
  return (
    <DomainColumns className="h-auto border-b bg-table-header py-[7px] font-medium text-gray-12">
      <span />
      <span>Domain</span>
      <span>Status</span>
      <span>Environment</span>
    </DomainColumns>
  );
}

const SKELETON_ROWS = ["a", "b", "c"];

function DomainsSkeleton() {
  return (
    <div aria-busy="true">
      <DomainsHeader />
      <ResourceListBody aria-hidden="true">
        {SKELETON_ROWS.map((key) => (
          <ResourceListItem key={key}>
            <DomainColumns>
              <Skeleton className="size-4 justify-self-center rounded-full" />
              <Skeleton className="h-3 w-48 max-w-full" />
              <Skeleton className="h-3 w-24 max-w-full" />
              <Skeleton className="h-3 w-20 max-w-full" />
            </DomainColumns>
          </ResourceListItem>
        ))}
      </ResourceListBody>
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
