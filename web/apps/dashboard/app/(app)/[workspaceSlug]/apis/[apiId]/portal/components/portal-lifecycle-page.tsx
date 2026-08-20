"use client";

import {
  type PortalState,
  portalQueryKey,
  usePortal,
  useUpdatePortal,
} from "@/lib/portal/use-portal";
import { useQueryClient } from "@tanstack/react-query";
import type { Portal } from "@unkey/api/models/components";
import { BookBookmark } from "@unkey/icons";
import {
  Button,
  PageBody,
  PageContainer,
  PageHeader,
  PageHeaderActions,
  PageHeaderContent,
  PageHeaderTitle,
} from "@unkey/ui";
import type { ReactNode } from "react";
import { useState } from "react";
import { CreatePortalDialog } from "./create-portal-dialog";
import { IntegrateDialog } from "./integrate-dialog";
import { PortalConfig } from "./portal-config";
import { SetupHero } from "./setup-hero";

// An API without a keyspace has nothing for a portal to serve keys from, so it
// is a dead end rather than a transient failure.
const NO_KEYSPACE_MESSAGE =
  "This API has no keyspace, so it cannot host a customer portal. Create a key for this API first.";

// A failed lookup is not the same dead end: the API may well have a keyspace we
// simply could not read, so this state keeps a retry action.
const KEYSPACE_LOOKUP_FAILED_MESSAGE =
  "We couldn't look up this API's keyspace. This is usually temporary — try again.";

type Props = {
  resourceName: string;
  /**
   * Undefined while `keyAuthIdLoading` or `keyAuthIdError` is set means
   * "unknown"; undefined with neither set means this API has no keyspace.
   */
  keyAuthId: string | undefined;
  keyAuthIdLoading: boolean;
  /** The keyspace lookup itself failed, which is retryable, not a dead end. */
  keyAuthIdError: boolean;
  onRetryKeyAuthId: () => void;
};

/**
 * Collapses the two independent resolutions the surface depends on — the
 * keyspace id and the portal itself — into the single state the page renders.
 */
function useSurfaceState(
  keyAuthId: string | undefined,
  keyAuthIdLoading: boolean,
  keyAuthIdError: boolean,
): PortalState {
  const portalState = usePortal(keyAuthId);

  if (keyAuthIdLoading) {
    return { status: "loading" };
  }
  // Ordered before the undefined check: a failed lookup also leaves the id
  // undefined, and reporting it as "no keyspace" would strand the operator on a
  // permanent message with no way back.
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
    <output
      aria-label="Loading customer portal"
      className="flex w-full items-center justify-center rounded-lg border border-grayA-4 p-12 text-[13px] text-gray-11"
    >
      Loading customer portal…
    </output>
  );
}

/**
 * Deliberately unlike the setup hero: a failed read must never read as
 * "no portal yet", because the two states offer opposite actions.
 */
function PortalErrorPanel({ message, onRetry }: { message: string; onRetry?: () => void }) {
  return (
    <div className="flex w-full flex-col gap-3 rounded-lg border border-error-6 bg-error-2 p-6">
      <h2 className="text-sm font-medium text-error-11">Couldn't load the customer portal</h2>
      <p className="text-[13px] leading-5 text-gray-11">{message}</p>
      {onRetry ? (
        <Button variant="outline" size="md" className="self-start" onClick={onRetry}>
          Retry
        </Button>
      ) : null}
    </div>
  );
}

function DisabledBanner({ onEnable, enabling }: { onEnable: () => void; enabling: boolean }) {
  return (
    <div className="flex flex-wrap items-center justify-between gap-3 rounded-lg border border-warning-6 bg-warning-2 p-4">
      <div className="space-y-1">
        <p className="text-sm font-medium text-gray-12">Portal disabled</p>
        <p className="text-[13px] text-gray-11">
          New sessions can't be created. Its configuration stays editable.
        </p>
      </div>
      <Button
        variant="primary"
        size="md"
        loading={enabling}
        loadingLabel="Enabling customer portal"
        onClick={onEnable}
      >
        Re-enable portal
      </Button>
    </div>
  );
}

/** Makes a missing `PortalState` variant a compile error rather than a blank body. */
function assertNever(value: never): never {
  throw new Error(`unhandled portal state: ${JSON.stringify(value)}`);
}

/**
 * The full customer-portal settings experience for a keyspace. Every state comes
 * from `getPortal`; a disabled portal renders the configuration view rather than
 * the setup hero, so its slug, branding, docs, and delete action stay reachable
 * without relaunching it first.
 */
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

  // A failed keyspace lookup retries the lookup; anything else retries the
  // portal read. Only a genuinely absent keyspace has nothing to retry.
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

  const renderBody = (): ReactNode => {
    switch (state.status) {
      case "loading":
        return <PortalLoading />;
      case "error":
        return <PortalErrorPanel message={state.message} onRetry={retry} />;
      case "notConfigured":
        return <SetupHero onEnable={() => setCreateOpen(true)} />;
      case "disabled":
      case "enabled":
        // `useSurfaceState` reports a missing keyspace as an error, so a
        // configured portal always has one; the panel keeps the impossible
        // case from rendering an empty body.
        return keyAuthId ? (
          <div className="flex w-full flex-col gap-6">
            {state.status === "disabled" && (
              <DisabledBanner
                enabling={updatePortal.isLoading}
                onEnable={() => setEnabled(state.portal, true)}
              />
            )}
            {/* The prop, not `state.portal.mapping.id`: a mapping id is only a
                keyspace id for keyspace mappings. */}
            <PortalConfig portal={state.portal} keyAuthId={keyAuthId} resourceName={resourceName} />
          </div>
        ) : (
          <PortalErrorPanel message={NO_KEYSPACE_MESSAGE} />
        );
      default:
        return assertNever(state);
    }
  };

  return (
    <PageContainer>
      {configuredPortal && (
        <PageHeader>
          <PageHeaderContent>
            <PageHeaderTitle>Customer portal</PageHeaderTitle>
          </PageHeaderContent>
          <PageHeaderActions>
            <Button variant="outline" onClick={() => setIntegrateOpen(true)}>
              <BookBookmark />
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
      {/* The snippets interpolate the portal's real slug, so the dialog only
          exists once a portal does. */}
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
