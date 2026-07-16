"use client";

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
import { useState } from "react";
import { DebugBar } from "./debug-bar";
import { IntegrateDialog } from "./integrate-dialog";
import { PortalConfig } from "./portal-config";
import { SetupHero } from "./setup-hero";
import { usePortalLifecycle } from "./use-portal-lifecycle";

/**
 * The full customer-portal settings experience for a keyspace. Enablement is
 * mocked per-resource in localStorage.
 */
export function PortalLifecyclePage({
  resourceId,
  resourceName,
}: {
  resourceId: string;
  resourceName: string;
}) {
  const { state, hydrated, enable, disable, forceState } = usePortalLifecycle(resourceId);
  const [integrateOpen, setIntegrateOpen] = useState(false);

  const url = "portal.unkey.com";
  const setup = state === "disabled" || state === "enabling";

  return (
    <PageContainer>
      {hydrated && !setup && (
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
      <PageBody>
        {hydrated &&
          (setup ? (
            <SetupHero enabling={state === "enabling"} onEnable={enable} />
          ) : (
            <PortalConfig
              resourceId={resourceId}
              resourceName={resourceName}
              url={url}
              onDisable={disable}
            />
          ))}
      </PageBody>
      <IntegrateDialog
        portalId={`portal_${resourceId}`}
        isOpen={integrateOpen}
        onOpenChange={setIntegrateOpen}
      />
      <DebugBar state={state} onSelect={forceState} />
    </PageContainer>
  );
}
