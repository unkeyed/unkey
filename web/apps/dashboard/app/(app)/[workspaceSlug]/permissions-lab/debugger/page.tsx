"use client";

import {
  Badge,
  Button,
  DialogContainer,
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
import { PermissionChip, UrnText } from "../components/urn-display";
import { WORKSPACE_ID, WORKSPACE_NAME, coverage, labelForPath } from "../lib/mock-data";
import { usePermissionsLab } from "../lib/store";
import { isPattern, parsePermission } from "../lib/urn";
import { type DebuggerQuery, analyzeAccess } from "./analysis";
import { QueryBar } from "./query-bar";
import { VerdictPanel } from "./verdict-panel";

/**
 * Lands on an interesting case: the support dashboard holds read-only grants,
 * so deleting a payments-prod key is denied with useful near-misses.
 */
const DEFAULT_QUERY: DebuggerQuery = {
  principalID: "unkey_root_support",
  resourcePath: "keyspaces/ks_payments_prod/keys/key_pay_001",
  action: "delete_key",
};

interface HistoryEntry {
  id: string;
  query: DebuggerQuery;
  allowed: boolean;
  at: number;
}

const timeFormat = new Intl.DateTimeFormat(undefined, {
  hour: "2-digit",
  minute: "2-digit",
  second: "2-digit",
});

export default function AccessDebuggerPage() {
  const lab = usePermissionsLab();
  const [draft, setDraft] = useState<DebuggerQuery>(DEFAULT_QUERY);
  const [submitted, setSubmitted] = useState<DebuggerQuery>(DEFAULT_QUERY);
  const [history, setHistory] = useState<HistoryEntry[]>([]);
  const [pendingGrant, setPendingGrant] = useState<string | null>(null);

  const principal = lab.state.principals.find((p) => p.id === submitted.principalID) ?? null;

  // Derived from live store state so the verdict flips the moment a fix is
  // granted, without requiring another explicit check.
  const analysis = useMemo(() => {
    if (!principal) {
      return null;
    }
    return analyzeAccess(
      principal,
      lab.state.roles,
      lab.effectivePermissions(principal.id),
      submitted,
    );
  }, [principal, lab, submitted]);

  const legacy = principal ? lab.legacyPermissions(principal.id) : [];

  const runCheck = (query: DebuggerQuery) => {
    const allowed = lab.can(query.principalID, {
      urn: { workspaceID: WORKSPACE_ID, resource: query.resourcePath },
      action: query.action,
    });
    setSubmitted(query);
    setHistory((prev) =>
      [{ id: crypto.randomUUID(), query, allowed, at: Date.now() }, ...prev].slice(0, 15),
    );
  };

  const rerun = (entry: HistoryEntry) => {
    setDraft(entry.query);
    runCheck(entry.query);
  };

  const confirmGrant = () => {
    if (pendingGrant === null || !principal) {
      return;
    }
    lab.commit("Grant via access debugger", [
      { op: "add", principalID: principal.id, permission: pendingGrant },
    ]);
    toast.success(`Granted to ${principal.name}`);
    setPendingGrant(null);
    // Record the re-check so the history shows the verdict flipping.
    runCheck(submitted);
  };

  const pendingDetails = useMemo(() => {
    if (pendingGrant === null) {
      return null;
    }
    const parsed = parsePermission(pendingGrant);
    if (!parsed.ok) {
      return null;
    }
    const path = parsed.value.urn.resource;
    return { pattern: isPattern(path), count: coverage(path).length };
  }, [pendingGrant]);

  const principalName = (principalID: string) =>
    lab.state.principals.find((p) => p.id === principalID)?.name ?? principalID;

  return (
    <PageContainer width="full">
      <PageHeader>
        <PageHeaderContent>
          <PageHeaderTitle>Access Debugger</PageHeaderTitle>
          <PageHeaderDescription>
            A REPL for authorization. Ask whether a root key can perform an action on a resource and
            get the exact grant that matched, or the near-misses explained.
          </PageHeaderDescription>
        </PageHeaderContent>
      </PageHeader>
      <PageBody>
        <div className="flex w-full flex-col gap-6">
          <QueryBar
            principals={lab.state.principals}
            query={draft}
            onChange={setDraft}
            onCheck={() => runCheck(draft)}
          />

          <div className="grid grid-cols-1 gap-6 xl:grid-cols-[minmax(0,2fr)_minmax(0,1fr)]">
            <div className="min-w-0">
              {principal && analysis ? (
                <VerdictPanel
                  analysis={analysis}
                  principalName={principal.name}
                  resourceLabel={labelForPath(submitted.resourcePath)}
                  action={submitted.action}
                  onRequestGrant={setPendingGrant}
                />
              ) : (
                <Empty>
                  <Empty.Title>No principal selected</Empty.Title>
                  <Empty.Description>
                    The checked root key no longer exists. Pick one in the query bar and run the
                    check again, or reset the lab data from the overview page.
                  </Empty.Description>
                </Empty>
              )}
            </div>

            <div className="flex min-w-0 flex-col gap-4">
              {principal && legacy.length > 0 && (
                <aside className="flex flex-col gap-2.5 rounded-lg border border-grayA-4 bg-grayA-2 p-4">
                  <span className="text-xs font-medium uppercase tracking-wide text-info-11">
                    Legacy permissions
                  </span>
                  <p className="text-xs text-gray-11">
                    {principal.name} also holds{" "}
                    {legacy.length === 1
                      ? "1 legacy tuple permission"
                      : `${legacy.length} legacy tuple permissions`}
                    . Legacy tuples match by exact string only, never by wildcard, so this check
                    ignores them.
                  </p>
                  <div className="flex flex-wrap gap-2">
                    {legacy.map((permission) => (
                      <PermissionChip key={permission} value={permission} legacy />
                    ))}
                  </div>
                </aside>
              )}

              <section className="flex flex-col rounded-lg border border-grayA-4">
                <header className="flex items-center justify-between border-b border-grayA-3 px-4 py-2.5">
                  <h2 className="text-xs uppercase tracking-wide text-gray-10">Session history</h2>
                  {history.length > 0 && (
                    <Button variant="ghost" size="sm" onClick={() => setHistory([])}>
                      Clear
                    </Button>
                  )}
                </header>
                {history.length === 0 ? (
                  <p className="px-4 py-6 text-xs text-gray-9">
                    No checks run yet this session. The page preloaded one for you; hit Check to
                    start recording, then click any row to re-run it.
                  </p>
                ) : (
                  <ul className="flex flex-col divide-y divide-grayA-3">
                    {history.map((entry) => (
                      <li key={entry.id}>
                        <button
                          type="button"
                          onClick={() => rerun(entry)}
                          className="flex w-full flex-col gap-1 px-4 py-2.5 text-left transition-colors hover:bg-grayA-2"
                        >
                          <span className="flex w-full items-center gap-2">
                            <Badge
                              variant={entry.allowed ? "success" : "error"}
                              size="sm"
                              className="uppercase"
                            >
                              {entry.allowed ? "allowed" : "denied"}
                            </Badge>
                            <span className="truncate text-xs text-gray-12">
                              {principalName(entry.query.principalID)}
                            </span>
                            <span className="ml-auto shrink-0 font-mono text-[10px] tabular-nums text-gray-9">
                              {timeFormat.format(entry.at)}
                            </span>
                          </span>
                          <span className="w-full truncate font-mono text-[11px] text-gray-10">
                            {entry.query.resourcePath}
                            <span className="text-gray-8">#</span>
                            {entry.query.action}
                          </span>
                        </button>
                      </li>
                    ))}
                  </ul>
                )}
              </section>
            </div>
          </div>
        </div>
      </PageBody>

      <DialogContainer
        isOpen={pendingGrant !== null}
        onOpenChange={(open) => {
          if (!open) {
            setPendingGrant(null);
          }
        }}
        title="Grant permission"
        subTitle="This adds a direct grant and takes effect immediately"
        footer={
          <div className="flex w-full items-center justify-end gap-3">
            <Button variant="ghost" onClick={() => setPendingGrant(null)}>
              Cancel
            </Button>
            <Button variant="primary" onClick={confirmGrant} disabled={!principal}>
              {principal ? `Grant to ${principal.name}` : "Grant"}
            </Button>
          </div>
        }
      >
        {pendingGrant !== null && (
          <div className="flex flex-col gap-3">
            <div className="overflow-x-auto rounded-lg border border-grayA-4 bg-grayA-2 px-3 py-2.5">
              <UrnText value={pendingGrant} />
            </div>
            <p className="text-sm text-gray-11">
              {pendingDetails?.pattern
                ? `This is a wildcard grant. It covers ${pendingDetails.count} existing ${
                    pendingDetails.count === 1 ? "resource" : "resources"
                  } in ${WORKSPACE_NAME}, plus any future ones that match the pattern.`
                : "This grants exactly one resource and one action, the narrowest possible fix."}
            </p>
            <p className="text-xs text-gray-10">
              Grants are allow-only: adding this permission can only widen access, and removing it
              later restores exactly the current state.
            </p>
          </div>
        )}
      </DialogContainer>
    </PageContainer>
  );
}
