"use client";

/**
 * Concept: share sheet on the resource. Instead of starting from a principal
 * and writing grants, start from a resource and manage who can touch it, the
 * way you would share a document or add a project member.
 */

import {
  Button,
  CopyButton,
  Empty,
  PageBody,
  PageContainer,
  PageHeader,
  PageHeaderContent,
  PageHeaderDescription,
  PageHeaderTitle,
  toast,
} from "@unkey/ui";
import { useMemo, useState } from "react";
import { resourceTypeOfPath } from "../lib/catalog";
import { ALL_RESOURCES, urnString } from "../lib/mock-data";
import { usePermissionsLab } from "../lib/store";
import { AccessList } from "./access-list";
import { type MatchedGrant, accessRowsForResource } from "./access-model";
import { GrantDialog } from "./grant-dialog";
import { ResourceRail } from "./resource-rail";

const DEFAULT_PATH = "keyspaces/ks_payments_prod";

export default function ShareSheetPage() {
  const lab = usePermissionsLab();
  const [selectedPath, setSelectedPath] = useState(DEFAULT_PATH);
  const [dialogOpen, setDialogOpen] = useState(false);

  const resource = useMemo(
    () => ALL_RESOURCES.find((r) => r.path === selectedPath) ?? null,
    [selectedPath],
  );

  const rows = useMemo(
    () =>
      resource ? accessRowsForResource(lab.state.principals, lab.state.roles, resource.path) : [],
    [lab.state.principals, lab.state.roles, resource],
  );

  const revoke = (principalID: string, grant: MatchedGrant) => {
    const principal = lab.state.principals.find((p) => p.id === principalID);
    const principalName = principal ? principal.name : principalID;
    const resourceLabel = resource ? resource.label : selectedPath;
    lab.commit(`Revoke ${grant.action} on ${resourceLabel} from ${principalName}`, [
      { op: "remove", principalID, permission: grant.permission },
    ]);
    toast.success(`Revoked ${grant.action} from ${principalName}`);
  };

  const typeLabel = resource ? (resourceTypeOfPath(resource.path)?.label ?? resource.type) : null;

  return (
    <PageContainer width="full">
      <PageHeader>
        <PageHeaderContent>
          <PageHeaderTitle>Share sheet on the resource</PageHeaderTitle>
          <PageHeaderDescription>
            Start from a resource and manage who can touch it, like sharing a doc. Direct grants are
            revocable right here; pattern and role access explain where they are managed.
          </PageHeaderDescription>
        </PageHeaderContent>
      </PageHeader>
      <PageBody>
        <div className="flex items-start gap-6">
          <ResourceRail selectedPath={selectedPath} onSelect={setSelectedPath} />

          {resource ? (
            <main className="min-w-0 grow flex flex-col gap-5">
              <div className="flex flex-col gap-1.5">
                <div className="flex items-center gap-2.5">
                  <h2 className="text-lg font-semibold text-accent-12">{resource.label}</h2>
                  {typeLabel && (
                    <span className="rounded border border-grayA-4 bg-grayA-2 px-1.5 py-0.5 text-[10px] uppercase tracking-wide text-gray-11">
                      {typeLabel}
                    </span>
                  )}
                </div>
                <div className="flex items-center gap-2 min-w-0">
                  <span className="truncate font-mono text-xs text-gray-11">
                    {urnString(resource.path)}
                  </span>
                  <CopyButton value={urnString(resource.path)} className="shrink-0" />
                </div>
              </div>

              <div className="rounded-lg border border-grayA-4">
                <div className="flex items-center justify-between gap-3 border-b border-grayA-4 px-4 py-3">
                  <div className="flex flex-col">
                    <span className="text-[13px] font-medium text-gray-12">Who has access</span>
                    <span className="text-xs text-gray-10">
                      {rows.length === 0
                        ? "No principals can reach this resource"
                        : `${rows.length} principal${rows.length === 1 ? "" : "s"}, direct and inherited`}
                    </span>
                  </div>
                  <Button variant="primary" onClick={() => setDialogOpen(true)}>
                    Grant access
                  </Button>
                </div>

                {rows.length === 0 ? (
                  <Empty className="py-12">
                    <Empty.Icon />
                    <Empty.Title>No one has access yet</Empty.Title>
                    <Empty.Description>
                      Nobody at ACME can touch {resource.label} right now. Grant a root key access
                      to get started.
                    </Empty.Description>
                    <Empty.Actions>
                      <Button variant="primary" onClick={() => setDialogOpen(true)}>
                        Grant access
                      </Button>
                    </Empty.Actions>
                  </Empty>
                ) : (
                  <AccessList rows={rows} onRevoke={revoke} />
                )}
              </div>

              <GrantDialog resource={resource} isOpen={dialogOpen} onOpenChange={setDialogOpen} />
            </main>
          ) : (
            <main className="min-w-0 grow">
              <Empty>
                <Empty.Icon />
                <Empty.Title>Pick a resource</Empty.Title>
                <Empty.Description>
                  Select a resource from the browser on the left to see who has access to it.
                </Empty.Description>
              </Empty>
            </main>
          )}
        </div>
      </PageBody>
    </PageContainer>
  );
}
