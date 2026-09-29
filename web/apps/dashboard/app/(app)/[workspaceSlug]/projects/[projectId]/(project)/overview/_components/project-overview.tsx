"use client";

import { useProject } from "@/hooks/use-project";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { projectDisplayName } from "@/lib/collections/deploy/projects";
import {
  IconBook2Outline18,
  IconChatsOutline18,
  IconGridOutline18,
  IconSquareTerminalOutline18,
} from "@unkey/icons";
import {
  EmptyState,
  EmptyStateDescription,
  EmptyStateHeader,
  EmptyStateIcon,
  EmptyStateTitle,
  PageBody,
  PageContainer,
  PageHeader,
  PageHeaderContent,
  PageHeaderTitle,
} from "@unkey/ui";
import { type ReactNode, useState } from "react";

const AGENT_PROMPT =
  "Set up Unkey in my project. Fetch https://unkey.com/agent/setup.md and follow it.";

export function ProjectOverviewPage() {
  const workspace = useWorkspaceNavigation();
  const { project } = useProject();

  return (
    <PageContainer>
      <PageHeader>
        <PageHeaderContent>
          <PageHeaderTitle>
            {project ? projectDisplayName(project, workspace.name) : ""}
          </PageHeaderTitle>
        </PageHeaderContent>
      </PageHeader>
      <PageBody className="flex flex-col gap-6">
        <EmptyState>
          <EmptyStateIcon>
            <IconGridOutline18 />
          </EmptyStateIcon>
          <EmptyStateHeader>
            <EmptyStateTitle>Project overview is coming soon</EmptyStateTitle>
            <EmptyStateDescription>
              A summary of your apps, services and activity will live here. Open an app to see its
              overview.
            </EmptyStateDescription>
          </EmptyStateHeader>
        </EmptyState>
        <HelpRow />
      </PageBody>
    </PageContainer>
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
    <section className="flex flex-col gap-2">
      <h2 className="text-[13px] font-medium text-gray-12">Need help?</h2>
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
    </section>
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
