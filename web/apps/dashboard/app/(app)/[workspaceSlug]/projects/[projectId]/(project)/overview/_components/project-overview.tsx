"use client";

import { useDeployActionGate } from "@/app/(app)/[workspaceSlug]/projects/_components/hooks/use-deploy-action-gate";
import { useAppHomeHref } from "@/hooks/use-app-home-href";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { routes } from "@/lib/navigation/routes";
import { trpc } from "@/lib/trpc/client";
import type { ProjectOverview } from "@/lib/trpc/routers/deploy/project/overview";
import { IconBook2Outline18, IconChatsOutline18, IconSquareTerminalOutline18 } from "@unkey/icons";
import {
  PageBody,
  PageContainer,
  PageHeader,
  PageHeaderActions,
  PageHeaderContent,
  PageHeaderTitle,
} from "@unkey/ui";
import { useParams, useRouter } from "next/navigation";
import { type ReactNode, useState } from "react";
import { Canvas, type CanvasActions, type CanvasLinks } from "./canvas";
import { type OverviewModel, buildOverviewModel } from "./overview-model";
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
        <Canvas data={data} model={model} links={links} actions={actions} />
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

function Section({ title, children }: { title: string; children: ReactNode }) {
  return (
    <section className="flex flex-col gap-2">
      <h2 className="text-[13px] font-medium text-gray-12">{title}</h2>
      {children}
    </section>
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
