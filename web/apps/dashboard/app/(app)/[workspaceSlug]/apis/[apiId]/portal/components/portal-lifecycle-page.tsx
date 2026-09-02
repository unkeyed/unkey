"use client";

import { useQueryClient } from "@tanstack/react-query";
import type { Portal } from "@unkey/api/models/components";
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
import {
  IconBookBookmarkOutline18,
  IconCircleWarningOutline18,
  IconTriangleWarningOutline18,
} from "nucleo-ui-outline-18";
import type { ReactNode } from "react";
import { useState } from "react";
import {
  type PortalState,
  portalQueryKey,
  usePortal,
  useUpdatePortal,
} from "@/lib/portal/use-portal";
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

function useSurfaceState(
  keyAuthId: string | undefined,
  keyAuthIdLoading: boolean,
  keyAuthIdError: boolean,
): PortalState {
  const portalState = usePortal(keyAuthId);

  if (keyAuthIdLoading) {
    return { status: "loading" };
  }
  // Must precede the undefined check: a failed lookup also leaves the id
  // undefined, and "no keyspace" offers no retry.
  if (keyAuthIdError) {
    return { status: "error", message: KEYSPACE_LOOKUP_FAILED_MESSAGE };
  }
  if (keyAuthId === undefined) {
    return { status: "error", message: NO_KEYSPACE_MESSAGE };
  }
  return portalState;
}

function PortalLoading() {
  return (
    <output aria-label="Loading customer portal" className="flex w-full flex-col gap-6">
      <Skeleton className="h-[320px] w-full rounded-lg" />
      <Skeleton className="h-[90px] w-full rounded-lg" />
    </output>
  );
}

function PortalErrorPanel({ message, onRetry }: { message: string; onRetry?: () => void }) {
  return (
    <AlertBanner variant="error">
      <IconCircleWarningOutline18 className="size-4" />
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

function DisabledBanner({ onEnable, enabling }: { onEnable: () => void; enabling: boolean }) {
  return (
    <AlertBanner variant="warning">
      <IconTriangleWarningOutline18 className="size-4" />
      <AlertBannerTitle>Portal disabled</AlertBannerTitle>
      <AlertBannerDescription>
        Your users can't sign in right now, but you can still change the settings below.
      </AlertBannerDescription>
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
  const state = useSurfaceState(keyAuthId, keyAuthIdLoading, keyAuthIdError);
  const queryClient = useQueryClient();
  const updatePortal = useUpdatePortal(keyAuthId ?? "");
  const [integrateOpen, setIntegrateOpen] = useState(false);
  const [createOpen, setCreateOpen] = useState(false);

  const configuredPortal =
    state.status === "enabled" || state.status === "disabled" ? state.portal : undefined;

  // Only a genuinely absent keyspace has nothing to retry.
  const retry = keyAuthIdError
    ? onRetryKeyAuthId
    : keyAuthId
      ? () => {
          void queryClient.invalidateQueries({ queryKey: portalQueryKey(keyAuthId) });
        }
      : undefined;

  const setEnabled = (portal: Portal, enabled: boolean) => {
    updatePortal.mutate({ portal: portal.id, enabled });
  };

  // `useSurfaceState` reports a missing keyspace as an error, so a configured
  // portal always has one; the fallback covers the impossible case.
  const renderConfigured = (portal: Portal, disabled: boolean): ReactNode =>
    keyAuthId ? (
      <div className="flex w-full flex-col gap-6">
        {disabled && (
          <DisabledBanner
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
            <PageHeaderTitle>Customer portal</PageHeaderTitle>
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
