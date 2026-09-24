"use client";

import { useDeployActionGate } from "@/app/(app)/[workspaceSlug]/projects/_components/hooks/use-deploy-action-gate";
import { useAppHomeHref } from "@/hooks/use-app-home-href";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import {
  DEPLOYMENT_STATUS_LABELS,
  deploymentStatusColor,
} from "@/lib/collections/deploy/deployment-status";
import { routes } from "@/lib/navigation/routes";
import { trpc } from "@/lib/trpc/client";
import type { ProjectOverview } from "@/lib/trpc/routers/deploy/project/overview";
import {
  IconBook2Outline18,
  IconChatsOutline18,
  IconCheckOutline18,
  IconCircleWarningOutline18,
  IconCodeBranchOutline18,
  IconSquareTerminalOutline18,
} from "@unkey/icons";
import {
  Button,
  PageBody,
  PageContainer,
  PageHeader,
  PageHeaderActions,
  PageHeaderContent,
  PageHeaderTitle,
} from "@unkey/ui";
import { cn } from "@unkey/ui/src/lib/utils";
import type { Route } from "next";
import Link from "next/link";
import { useParams, useRouter } from "next/navigation";
import { type ReactNode, useState } from "react";
import { Canvas, type CanvasActions, type CanvasLinks } from "./canvas";
import { type Attention, type OverviewModel, ago, buildOverviewModel } from "./overview-model";
import { ScenarioSwitcher } from "./scenario-switcher";

const AGENT_PROMPT =
  "Set up Unkey in my project. Fetch https://unkey.com/agent/setup.md and follow it.";

export function ProjectOverviewPage() {
  const params = useParams();
  const projectId = typeof params?.projectId === "string" ? params.projectId : "";
  const { data, isLoading, error } = trpc.deploy.project.overview.useQuery(
    { projectId },
    { enabled: Boolean(projectId), refetchInterval: 10_000 },
  );

  return (
    <PageContainer>
      {data ? (
        <Loaded data={data} />
      ) : (
        <PageBody>
          <div className="text-sm text-gray-9">
            {isLoading ? "Loading project…" : (error?.message ?? "No data")}
          </div>
        </PageBody>
      )}
      <ScenarioSwitcher />
    </PageContainer>
  );
}

function Loaded({ data }: { data: ProjectOverview }) {
  const router = useRouter();
  const workspace = useWorkspaceNavigation();
  const appHomeHref = useAppHomeHref();
  const { gated, openPaywall, planGate } = useDeployActionGate();
  const model = buildOverviewModel(data);
  const scope = { workspaceSlug: workspace.slug, projectId: data.project.id };

  const links: CanvasLinks = {
    app: (appId) => appHomeHref({ ...scope, appId }),
    allApps: routes.projects.detail(scope),
    keyspace: (apiId) => routes.apis.detail({ ...scope, apiId }),
    allKeyspaces: routes.apis.list(scope),
    ratelimit: (namespaceId) => routes.ratelimits.detail({ ...scope, namespaceId }),
    allRatelimits: routes.ratelimits.list(scope),
    logs: routes.projects.logs(scope),
  };
  const actions: CanvasActions = {
    createApp: () => (gated ? openPaywall() : router.push(routes.projects.apps.new(scope))),
    createKeyspace: () => router.push(routes.apis.list({ ...scope, new: true })),
    createRatelimit: () => router.push(routes.ratelimits.list(scope)),
  };

  return (
    <>
      <PageHeader>
        <PageHeaderContent>
          <PageHeaderTitle>{data.project.name}</PageHeaderTitle>
        </PageHeaderContent>
        <PageHeaderActions>
          <Summary data={data} model={model} />
        </PageHeaderActions>
      </PageHeader>
      <PageBody className="flex flex-col gap-6">
        {model.attention.length > 0 && (
          <AttentionStrip items={model.attention} links={links} logs={links.logs} />
        )}
        <Canvas data={data} model={model} links={links} actions={actions} />
        <div className="grid grid-cols-1 gap-6 lg:grid-cols-[1fr_320px]">
          {data.recentDeployments.length > 0 ? (
            <RecentDeployments data={data} links={links} />
          ) : (
            <FirstDeployHint shape={model.shape} onCreate={actions.createApp} />
          )}
          <SetupProgress model={model} />
        </div>
        <HelpRow />
      </PageBody>
      {planGate}
    </>
  );
}

function Summary({ data, model }: { data: ProjectOverview; model: OverviewModel }) {
  const parts = [
    data.apps.length && `${model.liveApps}/${data.apps.length} apps live`,
    data.keyspaces.length && `${data.keyspaces.length} keyspaces`,
    data.ratelimits.length && `${data.ratelimits.length} ratelimits`,
  ].filter(Boolean);
  return <span className="text-xs text-gray-9">{parts.join(" · ") || "Empty project"}</span>;
}

function AttentionStrip({
  items,
  links,
  logs,
}: {
  items: Attention[];
  links: CanvasLinks;
  logs: Route;
}) {
  const [first, ...rest] = items;
  const tone = first.kind === "failed" ? "error" : first.kind === "awaiting" ? "warning" : "info";
  const verb = {
    failed: "failed to deploy",
    awaiting: "is waiting for approval",
    building: "is deploying",
  }[first.kind];
  return (
    <div
      className={cn(
        "flex items-center gap-3 rounded-lg border px-3 py-2 text-[13px]",
        tone === "error" && "border-error-6 bg-error-2 text-error-12",
        tone === "warning" && "border-warning-6 bg-warning-2 text-warning-12",
        tone === "info" && "border-info-6 bg-info-2 text-info-12",
      )}
    >
      <IconCircleWarningOutline18 className="size-4 shrink-0" />
      <span className="min-w-0 truncate">
        <span className="font-medium">{first.app.name}</span> {verb}
        <span className="opacity-70"> · {ago(first.deployment.createdAt)}</span>
        {rest.length > 0 && <span className="opacity-70"> · {rest.length} more need a look</span>}
      </span>
      <span className="ml-auto flex shrink-0 gap-2">
        {first.kind === "failed" && (
          <Button size="sm" variant="outline" render={<Link href={logs} />}>
            View logs
          </Button>
        )}
        <Button size="sm" variant="outline" render={<Link href={links.app(first.app.id)} />}>
          Open {first.app.name}
        </Button>
      </span>
    </div>
  );
}

function Section({ title, children }: { title: string; children: ReactNode }) {
  return (
    <section className="flex flex-col gap-2">
      <h2 className="text-[13px] font-medium text-gray-12">{title}</h2>
      {children}
    </section>
  );
}

function RecentDeployments({ data, links }: { data: ProjectOverview; links: CanvasLinks }) {
  const appName = new Map(data.apps.map((a) => [a.id, a.name]));
  return (
    <Section title="Latest deployments">
      <div className="divide-y divide-border overflow-hidden rounded-lg border border-border bg-raised">
        {data.recentDeployments.map((d) => (
          <Link
            key={d.id}
            href={links.app(d.appId)}
            className="flex items-center gap-3 px-3 py-2 text-xs hover:bg-grayA-2"
          >
            <span
              className={cn("size-1.5 shrink-0 rounded-full", deploymentStatusColor(d.status))}
            />
            <span className="w-24 shrink-0 truncate font-medium text-gray-12">
              {appName.get(d.appId)}
            </span>
            <span className="min-w-0 flex-1 truncate text-gray-11">
              {d.commitMessage?.split("\n")[0] ?? "Image deployment"}
            </span>
            <span className="hidden shrink-0 items-center gap-1 font-mono text-gray-9 md:flex">
              <IconCodeBranchOutline18 className="size-3" />
              <span className="max-w-28 truncate">{d.branch ?? "main"}</span>
            </span>
            <span className="w-20 shrink-0 text-right text-gray-9">
              {DEPLOYMENT_STATUS_LABELS[d.status]}
            </span>
            <span className="w-14 shrink-0 text-right text-gray-9">{ago(d.createdAt)}</span>
          </Link>
        ))}
      </div>
    </Section>
  );
}

function FirstDeployHint({
  shape,
  onCreate,
}: {
  shape: OverviewModel["shape"];
  onCreate: () => void;
}) {
  return (
    <Section title="Latest deployments">
      <div className="flex items-center gap-4 rounded-lg border border-dashed border-grayA-6 px-4 py-5">
        <div className="min-w-0 flex-1">
          <div className="text-[13px] text-gray-12">No deployments yet</div>
          <div className="text-xs text-gray-9">
            {shape === "api"
              ? "Your keys are live. Deploy the service they protect and both sit in one project."
              : "Connect a repo and every push to main ships here."}
          </div>
        </div>
        <Button size="md" variant="outline" onClick={onCreate}>
          Create app
        </Button>
      </div>
    </Section>
  );
}

function SetupProgress({ model }: { model: OverviewModel }) {
  const done = model.steps.filter((s) => s.done).length;
  if (done === model.steps.length) {
    return <div />;
  }
  const next = model.steps.find((s) => !s.done);
  return (
    <Section title="Getting started">
      <div className="rounded-lg border border-border bg-raised p-3">
        <div className="mb-3 flex items-center gap-2">
          <div className="flex flex-1 gap-1">
            {model.steps.map((s) => (
              <span
                key={s.id}
                className={cn("h-1 flex-1 rounded-full", s.done ? "bg-success-9" : "bg-grayA-4")}
              />
            ))}
          </div>
          <span className="text-xs text-gray-9">
            {done}/{model.steps.length}
          </span>
        </div>
        <ul className="flex flex-col gap-1.5">
          {model.steps.map((s) => (
            <li key={s.id} className="flex items-center gap-2 text-xs">
              <span
                className={cn(
                  "flex size-4 items-center justify-center rounded-full border",
                  s.done
                    ? "border-success-9 bg-success-9 text-white"
                    : s === next
                      ? "border-gray-11"
                      : "border-grayA-6",
                )}
              >
                {s.done && <IconCheckOutline18 className="size-2.5" />}
              </span>
              <span
                className={cn(
                  s.done
                    ? "text-gray-9 line-through"
                    : s === next
                      ? "text-gray-12"
                      : "text-gray-11",
                )}
              >
                {s.label}
              </span>
              {s === next && <span className="ml-auto text-[11px] text-gray-9">Next</span>}
            </li>
          ))}
        </ul>
      </div>
    </Section>
  );
}

function HelpRow() {
  const [copied, setCopied] = useState(false);
  const copy = () => {
    navigator.clipboard?.writeText(AGENT_PROMPT);
    setCopied(true);
    window.setTimeout(() => setCopied(false), 1800);
  };
  const tile =
    "flex flex-col gap-2 rounded-lg border border-border bg-raised p-3 text-left transition-colors hover:border-strong";
  return (
    <Section title="Need help?">
      <div className="grid grid-cols-2 gap-3 md:grid-cols-4">
        <button type="button" onClick={copy} className={tile}>
          <IconSquareTerminalOutline18 className="size-4 text-gray-9" />
          <span>
            <span className="block text-[13px] font-medium text-gray-12">
              Set up with your agent
            </span>
            <span className="block text-xs text-gray-9">
              {copied ? "Prompt copied. Paste it into your agent." : "Claude, Cursor, Codex"}
            </span>
          </span>
        </button>
        <HelpLink
          href="https://unkey.com/discord"
          icon={<IconChatsOutline18 />}
          title="Ask in Discord"
          description="Community answers"
        />
        <HelpLink
          href="mailto:support@unkey.dev"
          icon={<IconChatsOutline18 />}
          title="Get support"
          description="Talk to an engineer"
        />
        <HelpLink
          href="https://unkey.com/docs"
          icon={<IconBook2Outline18 />}
          title="Documentation"
          description="Guides and API reference"
        />
      </div>
    </Section>
  );
}

function HelpLink({
  href,
  icon,
  title,
  description,
}: {
  href: string;
  icon: ReactNode;
  title: string;
  description: string;
}) {
  return (
    <a
      href={href}
      target="_blank"
      rel="noreferrer"
      className="flex flex-col gap-2 rounded-lg border border-border bg-raised p-3 transition-colors hover:border-strong"
    >
      <span className="text-gray-9 [&_svg]:size-4">{icon}</span>
      <span>
        <span className="block text-[13px] font-medium text-gray-12">{title}</span>
        <span className="block text-xs text-gray-9">{description}</span>
      </span>
    </a>
  );
}
