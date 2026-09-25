"use client";

import {
  IconArrowUpRightOutline12,
  IconCodeBranchOutline18,
  IconNodesOutline18,
  IconXmarkOutline18,
} from "@unkey/icons";
import { PageBody, PageContainer, PageHeader, PageHeaderContent, PageHeaderTitle } from "@unkey/ui";
import Link from "next/link";
import { type ReactNode, useEffect, useState } from "react";
import { ago } from "../../../overview/_components/overview-model";
import { ProtoCanvas, StatusLabel, sourceIcon } from "../proto-canvas";
import {
  HELP_LINKS,
  HealthDot,
  MonoLabel,
  type VariantProps,
  healthLine,
  num,
  pct,
  useCopyPrompt,
} from "../shared";

export function InspectorVariant(p: VariantProps) {
  const [selected, setSelected] = useState<string | null>(null);
  useEffect(() => {
    const onKey = (e: KeyboardEvent) => e.key === "Escape" && setSelected(null);
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, []);
  const app = p.data.apps.find((a) => a.id === selected) ?? null;
  return (
    <PageContainer>
      <PageHeader>
        <PageHeaderContent>
          <PageHeaderTitle>{p.title}</PageHeaderTitle>
        </PageHeaderContent>
      </PageHeader>
      <PageBody>
        <div className="flex h-[600px] gap-3">
          <div className="min-w-0 flex-1">
            <ProtoCanvas
              {...p}
              options={{
                heightClass: "h-full",
                selectedAppId: selected,
                onSelectApp: (id) => setSelected((s) => (s === id ? null : id)),
              }}
            />
          </div>
          <aside className="flex w-[300px] shrink-0 flex-col overflow-y-auto rounded-xl border border-border bg-raised">
            {app ? (
              <AppPanel p={p} appId={app.id} onClose={() => setSelected(null)} />
            ) : (
              <ProjectPanel p={p} />
            )}
          </aside>
        </div>
      </PageBody>
    </PageContainer>
  );
}

function Field({ label, children }: { label: string; children: ReactNode }) {
  return (
    <div className="flex flex-col gap-1">
      <MonoLabel>{label}</MonoLabel>
      <div className="min-w-0 text-[13px] text-gray-12">{children}</div>
    </div>
  );
}

function AppPanel({ p, appId, onClose }: { p: VariantProps; appId: string; onClose: () => void }) {
  const app = p.data.apps.find((a) => a.id === appId);
  if (!app) {
    return null;
  }
  const d = app.latest;
  const keyspaces = p.data.keyspaces.filter((k) =>
    p.data.keyspaceLinks.some((l) => l.appId === app.id && l.keyAuthId === k.keyAuthId),
  );
  return (
    <div className="flex flex-col">
      <div className="flex items-center gap-2 border-b border-border px-4 py-3">
        <span className="text-gray-11 [&_svg]:size-4">{sourceIcon(app)}</span>
        <span className="min-w-0 truncate text-sm font-medium text-gray-12">{app.name}</span>
        <button
          type="button"
          onClick={onClose}
          aria-label="Close"
          className="ml-auto rounded-md p-1 text-gray-9 hover:bg-grayA-3 hover:text-gray-12"
        >
          <IconXmarkOutline18 className="size-3.5" />
        </button>
      </div>
      <div className="flex flex-col gap-5 px-4 py-4">
        <Field label="Status">
          <StatusLabel app={app} />
        </Field>
        {app.domain && (
          <Field label="Domain">
            <a
              href={`https://${app.domain}`}
              target="_blank"
              rel="noreferrer"
              className="flex items-center gap-1 truncate font-mono text-xs text-gray-11 hover:text-gray-12"
            >
              {app.domain}
              <IconArrowUpRightOutline12 className="size-3 shrink-0" />
            </a>
          </Field>
        )}
        {app.repositoryFullName && (
          <Field label="Repository">
            <span className="font-mono text-xs text-gray-11">{app.repositoryFullName}</span>
          </Field>
        )}
        {d && (
          <Field label="Latest deploy">
            <div className="flex flex-col gap-1.5 rounded-lg border border-border p-2.5">
              <span className="flex items-center gap-1.5 text-xs text-gray-11">
                <IconCodeBranchOutline18 className="size-3.5 text-gray-9" />
                <span className="font-mono">{d.branch ?? "main"}</span>
                {d.commitSha && (
                  <span className="font-mono text-gray-9">{d.commitSha.slice(0, 7)}</span>
                )}
                <span className="ml-auto text-gray-9">{ago(d.createdAt)}</span>
              </span>
              <span className="text-xs text-gray-12">
                {d.commitMessage?.split("\n")[0] ?? "Image deployment"}
              </span>
              {d.author && <span className="text-[11px] text-gray-9">by {d.author}</span>}
            </div>
          </Field>
        )}
        <Field label="Verifies keys in">
          {keyspaces.length ? (
            <div className="flex flex-col gap-1">
              {keyspaces.map((k) => (
                <Link
                  key={k.apiId}
                  href={p.links.keyspace(k.apiId)}
                  className="flex items-center gap-2 text-xs text-gray-11 hover:text-gray-12"
                >
                  <IconNodesOutline18 className="size-3.5 text-gray-9" />
                  {k.name}
                  <span className="ml-auto tabular-nums text-gray-9">{num(k.keyCount)} keys</span>
                </Link>
              ))}
            </div>
          ) : (
            <span className="text-xs text-gray-9">No keyspace linked in the gateway config</span>
          )}
        </Field>
        <Link
          href={p.links.app(app.id)}
          className="flex h-8 items-center justify-center rounded-md bg-gray-12 text-[13px] font-medium text-gray-1 hover:bg-gray-11"
        >
          Open app
        </Link>
      </div>
    </div>
  );
}

function ProjectPanel({ p }: { p: VariantProps }) {
  const { copied, copy } = useCopyPrompt();
  const t = p.totals;
  return (
    <div className="flex h-full flex-col gap-5 px-4 py-4">
      <div className="flex flex-col gap-1">
        <MonoLabel>Project</MonoLabel>
        <span className="flex items-center gap-2 text-xs text-gray-11">
          <HealthDot totals={t} />
          {healthLine(t, p.data.apps.length)}
        </span>
      </div>
      <div className="grid grid-cols-2 gap-4">
        <Field label="Keys">{num(t.keys)}</Field>
        <Field label="Verified · 7d">{num(t.verified)}</Field>
        <Field label="Ratelimited · 7d">{num(t.rlRequests)}</Field>
        <Field label="Blocked">{pct(t.rlBlockedPct)}</Field>
      </div>
      <p className="text-xs text-gray-9">
        Select an app on the canvas to see its deploy and the keys it checks.
      </p>
      <div className="mt-auto flex flex-col gap-1 border-t border-border pt-4">
        <MonoLabel className="mb-1">Help</MonoLabel>
        <button
          type="button"
          onClick={copy}
          className="py-1 text-left text-[13px] text-gray-11 hover:text-gray-12"
        >
          {copied ? "Prompt copied" : "Set up with your agent"}
        </button>
        {HELP_LINKS.map((l) => (
          <a
            key={l.title}
            href={l.href}
            target="_blank"
            rel="noreferrer"
            className="py-1 text-[13px] text-gray-11 hover:text-gray-12"
          >
            {l.title}
          </a>
        ))}
      </div>
    </div>
  );
}
