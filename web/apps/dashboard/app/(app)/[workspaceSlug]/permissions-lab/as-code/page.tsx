"use client";

/**
 * Concept 5: Permissions as Code, Two-Way. The same direct grants rendered as
 * a visual chip editor and as a plain-text document, always in sync, with the
 * exact public API call for every edit shown underneath.
 */

import {
  Empty,
  PageBody,
  PageContainer,
  PageHeader,
  PageHeaderContent,
  PageHeaderDescription,
  PageHeaderTitle,
  Tabs,
  TabsList,
  TabsTrigger,
} from "@unkey/ui";
import { useMemo, useState } from "react";
import { type GrantOp, usePermissionsLab } from "../lib/store";
import { ApiEquivalent, type ChangeDiff } from "./api-equivalent";
import { CodePane, type DraftAnalysis, analyzeDraft } from "./code-pane";
import { VisualEditor } from "./visual-editor";

interface Draft {
  text: string;
  /** canonical text at the moment the draft started, for conflict detection */
  base: string;
}

const EMPTY_ANALYSIS: DraftAnalysis = { errors: [], added: [], removed: [] };

export default function AsCodePage() {
  const lab = usePermissionsLab();
  const principals = lab.state.principals;

  const [selectedID, setSelectedID] = useState<string | null>(null);
  const [drafts, setDrafts] = useState<Record<string, Draft>>({});
  const [lastChange, setLastChange] = useState<{
    principalID: string;
    diff: ChangeDiff;
  } | null>(null);

  const principal = principals.find((p) => p.id === selectedID) ?? principals[0];

  const direct = useMemo(() => (principal ? [...principal.permissions].sort() : []), [principal]);
  const canonicalText = direct.join("\n");
  const draft: Draft | null = principal ? (drafts[principal.id] ?? null) : null;
  const analysis = useMemo(
    () => (draft ? analyzeDraft(draft.text, direct) : null),
    [draft, direct],
  );

  if (!principal) {
    return (
      <PageContainer width="full">
        <PageBody>
          <Empty>
            <Empty.Title>No root keys</Empty.Title>
            <Empty.Description>
              The lab data has no principals. Reset the lab data from the overview page.
            </Empty.Description>
          </Empty>
        </PageBody>
      </PageContainer>
    );
  }

  const roles = principal.roles.flatMap((roleID) => {
    const role = lab.state.roles.find((r) => r.id === roleID);
    return role ? [role] : [];
  });

  const commitDiff = (title: string, diff: ChangeDiff) => {
    const ops: GrantOp[] = [
      ...diff.added.map((permission) => ({
        op: "add" as const,
        principalID: principal.id,
        permission,
      })),
      ...diff.removed.map((permission) => ({
        op: "remove" as const,
        principalID: principal.id,
        permission,
      })),
    ];
    lab.commit(title, ops);
    setLastChange({ principalID: principal.id, diff });
  };

  const clearDraft = () => {
    setDrafts((prev) => {
      const next = { ...prev };
      delete next[principal.id];
      return next;
    });
  };

  const pending: ChangeDiff | null =
    draft !== null && analysis !== null && analysis.errors.length === 0
      ? { added: analysis.added, removed: analysis.removed }
      : null;

  return (
    <PageContainer width="full">
      <PageHeader>
        <PageHeaderContent>
          <PageHeaderTitle>Permissions as Code, Two-Way</PageHeaderTitle>
          <PageHeaderDescription>
            The same grants as a visual editor and a text document, always in sync. Everything you
            click is an API call you could have made yourself.
          </PageHeaderDescription>
        </PageHeaderContent>
      </PageHeader>
      <PageBody>
        <div className="flex flex-col gap-4">
          <div className="flex flex-col gap-2">
            <div className="overflow-x-auto">
              <Tabs
                value={principal.id}
                onValueChange={(value) => {
                  if (typeof value === "string") {
                    setSelectedID(value);
                  }
                }}
              >
                <TabsList aria-label="Root keys">
                  {principals.map((p) => (
                    <TabsTrigger key={p.id} value={p.id}>
                      {p.name}
                    </TabsTrigger>
                  ))}
                </TabsList>
              </Tabs>
            </div>
            <div className="flex items-baseline gap-3 text-xs text-gray-10">
              <span className="font-mono">{principal.id}</span>
              <span>created {principal.createdAt}</span>
              <span>
                {direct.length} direct grant{direct.length === 1 ? "" : "s"}
              </span>
              <span>
                {roles.length} role{roles.length === 1 ? "" : "s"}
              </span>
            </div>
          </div>

          <div className="grid grid-cols-1 xl:grid-cols-2 gap-4 items-start">
            <VisualEditor
              principal={principal}
              roles={roles}
              legacy={lab.legacyPermissions(principal.id)}
              onAdd={(permission) =>
                commitDiff(`Grant ${principal.name}`, { added: [permission], removed: [] })
              }
              onRemove={(permission) =>
                commitDiff(`Revoke from ${principal.name}`, { added: [], removed: [permission] })
              }
            />
            <CodePane
              text={draft !== null ? draft.text : canonicalText}
              dirty={draft !== null}
              conflict={draft !== null && draft.base !== canonicalText}
              analysis={analysis ?? EMPTY_ANALYSIS}
              onChange={(text) =>
                setDrafts((prev) => ({
                  ...prev,
                  [principal.id]: {
                    text,
                    base: prev[principal.id]?.base ?? canonicalText,
                  },
                }))
              }
              onApply={() => {
                if (pending === null) {
                  return;
                }
                commitDiff(`Edit ${principal.name} as code`, pending);
                clearDraft();
              }}
              onRevert={clearDraft}
            />
          </div>

          <ApiEquivalent
            keyId={principal.id}
            currentPermissions={direct}
            pending={pending}
            lastApplied={lastChange?.principalID === principal.id ? lastChange.diff : null}
          />
        </div>
      </PageBody>
    </PageContainer>
  );
}
