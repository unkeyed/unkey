"use client";

/**
 * The DX statement piece: the exact public API call behind whatever the UI is
 * about to do (pending code edit), just did (last applied change), or would
 * need to do to recreate the current grant set from scratch.
 */

import { Code, CopyButton, Tabs, TabsContent, TabsList, TabsTrigger } from "@unkey/ui";

export interface ChangeDiff {
  added: string[];
  removed: string[];
}

function isEmpty(diff: ChangeDiff): boolean {
  return diff.added.length === 0 && diff.removed.length === 0;
}

function curlCall(
  method: "addPermissions" | "removePermissions",
  keyId: string,
  permissions: string[],
): string {
  const body = JSON.stringify({ keyId, permissions }, null, 2);
  return [
    `curl -X POST https://api.unkey.com/v2/keys.${method} \\`,
    '  -H "Authorization: Bearer <UNKEY_ROOT_KEY>" \\',
    '  -H "Content-Type: application/json" \\',
    `  -d '${body}'`,
  ].join("\n");
}

function curlSnippet(keyId: string, diff: ChangeDiff): string {
  const calls: string[] = [];
  if (diff.added.length > 0) {
    calls.push(curlCall("addPermissions", keyId, diff.added));
  }
  if (diff.removed.length > 0) {
    calls.push(curlCall("removePermissions", keyId, diff.removed));
  }
  return calls.join("\n\n");
}

function sdkCall(
  method: "addPermissions" | "removePermissions",
  keyId: string,
  permissions: string[],
): string {
  const list = permissions.map((p) => `    ${JSON.stringify(p)},`).join("\n");
  return [
    `await unkey.keys.${method}({`,
    `  keyId: ${JSON.stringify(keyId)},`,
    "  permissions: [",
    list,
    "  ],",
    "});",
  ].join("\n");
}

function sdkSnippet(keyId: string, diff: ChangeDiff): string {
  const calls: string[] = [];
  if (diff.added.length > 0) {
    calls.push(sdkCall("addPermissions", keyId, diff.added));
  }
  if (diff.removed.length > 0) {
    calls.push(sdkCall("removePermissions", keyId, diff.removed));
  }
  return [
    'import { Unkey } from "@unkey/api";',
    "",
    "const unkey = new Unkey({ rootKey: process.env.UNKEY_ROOT_KEY });",
    "",
    calls.join("\n\n"),
  ].join("\n");
}

export function ApiEquivalent({
  keyId,
  currentPermissions,
  pending,
  lastApplied,
}: {
  keyId: string;
  /** the full current direct grant set, canonical order */
  currentPermissions: string[];
  /** valid unapplied diff from the code pane, if any */
  pending: ChangeDiff | null;
  /** the most recent applied change for this principal, if any */
  lastApplied: ChangeDiff | null;
}) {
  let label: string;
  let diff: ChangeDiff;
  if (pending !== null && !isEmpty(pending)) {
    label = "Reproduces the pending code edit";
    diff = pending;
  } else if (lastApplied !== null && !isEmpty(lastApplied)) {
    label = "Reproduces the last applied change";
    diff = lastApplied;
  } else {
    label = "Recreates the full current grant set";
    diff = { added: currentPermissions, removed: [] };
  }

  const summary = [
    diff.added.length > 0
      ? `+${diff.added.length} grant${diff.added.length === 1 ? "" : "s"}`
      : null,
    diff.removed.length > 0
      ? `-${diff.removed.length} grant${diff.removed.length === 1 ? "" : "s"}`
      : null,
  ]
    .filter((s) => s !== null)
    .join(", ");

  return (
    <section className="rounded-lg border border-grayA-4 p-4 flex flex-col gap-3 min-w-0">
      <header className="flex items-baseline justify-between gap-3 flex-wrap">
        <div className="flex flex-col gap-0.5">
          <h2 className="text-sm font-medium text-gray-12">API equivalent</h2>
          <p className="text-xs text-gray-10">
            This page never does anything you could not do with a root key and curl.
          </p>
        </div>
        {!isEmpty(diff) && (
          <span className="text-xs text-gray-11">
            {label}
            <span className="font-mono text-gray-9"> ({summary})</span>
          </span>
        )}
      </header>

      {isEmpty(diff) ? (
        <p className="text-sm text-gray-10 rounded-md bg-grayA-2 px-3 py-2">
          No direct grants yet. Add one above and the exact API call appears here.
        </p>
      ) : (
        <Tabs defaultValue="curl">
          <TabsList>
            <TabsTrigger value="curl">cURL</TabsTrigger>
            <TabsTrigger value="ts">TypeScript SDK</TabsTrigger>
          </TabsList>
          <TabsContent value="curl">
            <Code
              copyButton={<CopyButton value={curlSnippet(keyId, diff)} />}
              preClassName="overflow-x-auto"
            >
              {curlSnippet(keyId, diff)}
            </Code>
          </TabsContent>
          <TabsContent value="ts">
            <Code
              copyButton={<CopyButton value={sdkSnippet(keyId, diff)} />}
              preClassName="overflow-x-auto"
            >
              {sdkSnippet(keyId, diff)}
            </Code>
          </TabsContent>
        </Tabs>
      )}
    </section>
  );
}
