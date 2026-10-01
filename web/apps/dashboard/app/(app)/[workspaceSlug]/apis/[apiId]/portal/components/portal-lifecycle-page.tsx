"use client";
import { queryKeys } from "@/lib/query-keys";

import { type PortalState, usePortal, useUpdatePortal } from "@/lib/portal/use-portal";
import { useQueryClient } from "@tanstack/react-query";
import type { Portal } from "@unkey/api/models/components";
import {
  IconBookBookmarkOutline18,
  IconCircleWarningOutline18,
  IconTriangleWarningOutline18,
} from "@unkey/icons";
import { match } from "@unkey/match";
import {
  AlertBanner,
  AlertBannerActions,
  AlertBannerDescription,
  AlertBannerTitle,
  Button,
  PageBody,
  PageContainer,
  PageHeader,
  PageHeaderActions,
  PageHeaderContent,
  PageHeaderTitle,
  Skeleton,
} from "@unkey/ui";
import type { ReactNode } from "react";
import { useState } from "react";
import { CreatePortalDialog } from "./create-portal-dialog";
import { IntegrateDialog } from "./integrate-dialog";
import { PortalConfig } from "./portal-config";
import { SetupHero } from "./setup-hero";

// A dead end, not a transient failure: there is nothing to retry.
const NO_KEYSPACE_MESSAGE =
  "This API has no keyspace, so a portal would have no keys to show. Create a key for this API first.";

// A failed lookup is retryable: the API may have a keyspace we could not read.
const KEYSPACE_LOOKUP_FAILED_MESSAGE =
  "We couldn't look up this API's keyspace. This is usually temporary, so try again.";

type Props = {
  resourceName: string;
  /**
   * Undefined while `keyAuthIdLoading` or `keyAuthIdError` is set means
   * "unknown"; undefined with neither set means this API has no keyspace.
   */
  keyAuthId: string | undefined;
  keyAuthIdLoading: boolean;
  keyAuthIdError: boolean;
  onRetryKeyAuthId: () => void;
};

export function usePortalSurfaceState(
  keyAuthId: string | undefined,
  keyAuthIdLoading: boolean,
  keyAuthIdError: boolean,
  onRetryKeyAuthId: () => void,
): { state: PortalState; retry: (() => void) | undefined } {
  const portalState = usePortal(keyAuthId);
  const queryClient = useQueryClient();

  if (keyAuthIdLoading) {
    return { state: { status: "loading" }, retry: undefined };
  }
  // Must precede the undefined check: a failed lookup also leaves the id
  // undefined, and "no keyspace" offers no retry.
  if (keyAuthIdError) {
    return {
      state: { status: "error", message: KEYSPACE_LOOKUP_FAILED_MESSAGE },
      retry: onRetryKeyAuthId,
    };
  }
  if (keyAuthId === undefined) {
    return { state: { status: "error", message: NO_KEYSPACE_MESSAGE }, retry: undefined };
  }
  return {
    state: portalState,
    retry: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.portal.detail(keyAuthId) });
    },
  };
}

function PortalLoading() {
  return (
    <output aria-label="Loading customer portal" className="flex w-full flex-col gap-6">
      <Skeleton className="h-[320px] w-full rounded-lg" />
      <Skeleton className="h-[90px] w-full rounded-lg" />
    </output>
  );
}

export function PortalErrorPanel({ message, onRetry }: { message: string; onRetry?: () => void }) {
  return (
    <AlertBanner variant="error">
      <IconCircleWarningOutline18 className="size-3.5" />
      <AlertBannerTitle>Couldn't load the customer portal</AlertBannerTitle>
      <AlertBannerDescription>{message}</AlertBannerDescription>
      {onRetry ? (
        <AlertBannerActions>
          <Button variant="outline" size="md" onClick={onRetry}>
            Retry
          </Button>
        </AlertBannerActions>
      ) : null}
    </AlertBanner>
  );
}

export function DisabledBanner({
  description,
  onEnable,
  enabling,
}: {
  description: string;
  onEnable: () => void;
  enabling: boolean;
}) {
  return (
    <AlertBanner variant="warning">
      <IconTriangleWarningOutline18 className="size-3.5" />
      <AlertBannerTitle>Portal disabled</AlertBannerTitle>
      <AlertBannerDescription>{description}</AlertBannerDescription>
      <AlertBannerActions>
        <Button
          variant="primary"
          size="md"
          loading={enabling}
          loadingLabel="Enabling customer portal"
          onClick={onEnable}
        >
          Re-enable portal
        </Button>
      </AlertBannerActions>
    </AlertBanner>
  );
}

// A disabled portal renders the configuration view rather than the setup hero,
// so its slug, branding, and delete action stay reachable without re-enabling.
export function PortalLifecyclePage({
  resourceName,
  keyAuthId,
  keyAuthIdLoading,
  keyAuthIdError,
  onRetryKeyAuthId,
}: Props) {
  const { state, retry } = usePortalSurfaceState(
    keyAuthId,
    keyAuthIdLoading,
    keyAuthIdError,
    onRetryKeyAuthId,
  );
  const updatePortal = useUpdatePortal(keyAuthId ?? "");
  const [integrateOpen, setIntegrateOpen] = useState(false);
  const [createOpen, setCreateOpen] = useState(false);

  const configuredPortal =
    state.status === "enabled" || state.status === "disabled" ? state.portal : undefined;

  const setEnabled = (portal: Portal, enabled: boolean) => {
    updatePortal.mutate({ portal: portal.id, enabled });
  };

  // `usePortalSurfaceState` reports a missing keyspace as an error, so a configured
  // portal always has one; the fallback covers the impossible case.
  const renderConfigured = (portal: Portal, disabled: boolean): ReactNode =>
    keyAuthId ? (
      <div className="flex w-full flex-col gap-6">
        {disabled && (
          <DisabledBanner
            description="Your users can't sign in right now, but you can still change the settings below."
            enabling={updatePortal.isLoading}
            onEnable={() => setEnabled(portal, true)}
          />
        )}
        <PortalConfig portal={portal} keyAuthId={keyAuthId} />
      </div>
    ) : (
      <PortalErrorPanel message={NO_KEYSPACE_MESSAGE} />
    );

  const renderBody = (): ReactNode =>
    match(state)
      .with({ status: "loading" }, () => <PortalLoading />)
      .with({ status: "error" }, ({ message }) => (
        <PortalErrorPanel message={message} onRetry={retry} />
      ))
      .with({ status: "notConfigured" }, () => <SetupHero onEnable={() => setCreateOpen(true)} />)
      .with({ status: "disabled" }, ({ portal }) => renderConfigured(portal, true))
      .with({ status: "enabled" }, ({ portal }) => renderConfigured(portal, false))
      .exhaustive();

  return (
    <PageContainer>
      {configuredPortal && (
        <PageHeader>
          <PageHeaderContent>
            <PageHeaderTitle>Portal settings</PageHeaderTitle>
          </PageHeaderContent>
          <PageHeaderActions>
            <Button variant="outline" onClick={() => setIntegrateOpen(true)}>
              <IconBookBookmarkOutline18 />
              Integration docs
            </Button>
          </PageHeaderActions>
        </PageHeader>
      )}
      <PageBody>{renderBody()}</PageBody>
      {/* Mounted only while open so each run prefills a fresh slug candidate. */}
      {createOpen && keyAuthId ? (
        <CreatePortalDialog
          keyAuthId={keyAuthId}
          resourceName={resourceName}
          isOpen={createOpen}
          onOpenChange={setCreateOpen}
        />
      ) : null}
      {/* The snippets interpolate the portal's real slug. */}
      {configuredPortal ? (
        <IntegrateDialog
          slug={configuredPortal.slug}
          isOpen={integrateOpen}
          onOpenChange={setIntegrateOpen}
        />
      ) : null}
    </PageContainer>
  );
}
