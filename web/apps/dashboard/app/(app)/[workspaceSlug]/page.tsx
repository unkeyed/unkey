"use client";

import { Navbar } from "@/components/navigation/navbar";
import { ProximityPrefetch } from "@/components/proximity-prefetch";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { formatNumber } from "@/lib/fmt";
import { CodeBranch, Cube, Dots, Gauge, Key, TriangleWarning2 } from "@unkey/icons";
import { Button, TimestampInfo } from "@unkey/ui";
import Link from "next/link";
import type React from "react";
import { useMemo } from "react";
import { Avatar } from "./projects/[projectId]/apps/[appId]/components/git-avatar";
import { ProjectActions } from "./projects/_components/list/project-actions";
import { ResourceCard } from "./projects/_components/list/resource-card";

// Demo / showcase data. The workspace overview page is rendered against
// these constants instead of live queries so the layout is reviewable on a
// fresh clone without seeding MySQL or ClickHouse. Replace with real queries
// before shipping.

const VISIBLE_ROWS = 3;

const HOUR_MS = 3_600_000;
const DAY_MS = 24 * HOUR_MS;

type DemoProject = {
  id: string;
  name: string;
  domain: string | null;
  commitTitle: string;
  branch: string;
  author: string;
  authorAvatar: string;
  commitTimestampOffsetMs: number;
};

const DEMO_PROJECTS: DemoProject[] = [
  {
    id: "proj_marketing",
    name: "marketing-site",
    domain: "acme.com",
    commitTitle: "feat: add June 5 changelog entry for keyspaces",
    branch: "main",
    author: "chronark",
    authorAvatar: "https://avatars.githubusercontent.com/u/4798786?v=4",
    commitTimestampOffsetMs: 2 * HOUR_MS,
  },
  {
    id: "proj_docs",
    name: "docs",
    domain: "docs.acme.com",
    commitTitle: "fix: broken anchor links on /docs/pricing",
    branch: "main",
    author: "perkinsjr",
    authorAvatar: "https://avatars.githubusercontent.com/u/8939141?v=4",
    commitTimestampOffsetMs: 6 * HOUR_MS,
  },
  {
    id: "proj_dashboard",
    name: "dashboard",
    domain: "app.acme.com",
    commitTitle: "feat: rebuild workspace overview to match Vercel layout",
    branch: "main",
    author: "chronark",
    authorAvatar: "https://avatars.githubusercontent.com/u/4798786?v=4",
    commitTimestampOffsetMs: 1 * DAY_MS,
  },
  {
    id: "proj_api_gateway",
    name: "api-gateway",
    domain: "api.acme.com",
    commitTitle: "perf: cut /verify p99 from 11ms to 6ms via pool reuse",
    branch: "release/2026",
    author: "mike-stewart",
    authorAvatar: "https://avatars.githubusercontent.com/u/13297017?v=4",
    commitTimestampOffsetMs: 2 * DAY_MS,
  },
  {
    id: "proj_auth",
    name: "auth-service",
    domain: "auth.acme.com",
    commitTitle: "fix: token refresh race when two requests share a session",
    branch: "main",
    author: "domeccleston",
    authorAvatar: "https://avatars.githubusercontent.com/u/23363839?v=4",
    commitTimestampOffsetMs: 3 * DAY_MS,
  },
  {
    id: "proj_analytics",
    name: "analytics",
    domain: "analytics.acme.com",
    commitTitle: "feat: stream deployment events to clickhouse",
    branch: "main",
    author: "chronark",
    authorAvatar: "https://avatars.githubusercontent.com/u/4798786?v=4",
    commitTimestampOffsetMs: 5 * DAY_MS,
  },
  {
    id: "proj_webhooks",
    name: "webhooks-worker",
    domain: "hooks.acme.com",
    commitTitle: "chore: bump go to 1.24 and re-vendor deps",
    branch: "main",
    author: "perkinsjr",
    authorAvatar: "https://avatars.githubusercontent.com/u/8939141?v=4",
    commitTimestampOffsetMs: 7 * DAY_MS,
  },
  {
    id: "proj_playground",
    name: "playground",
    domain: "play.acme.com",
    commitTitle: "feat: live key tester with copyable curl",
    branch: "main",
    author: "mike-stewart",
    authorAvatar: "https://avatars.githubusercontent.com/u/13297017?v=4",
    commitTimestampOffsetMs: 14 * DAY_MS,
  },
];

type DemoKeyspace = { id: string; name: string; keyCount: number };

const DEMO_KEYSPACES: DemoKeyspace[] = [
  { id: "api_payments", name: "Payments API", keyCount: 142 },
  { id: "api_customer", name: "Customer API", keyCount: 38 },
  { id: "api_internal", name: "Internal API", keyCount: 7 },
  { id: "api_public", name: "Public API", keyCount: 891 },
  { id: "api_webhooks", name: "Webhooks", keyCount: 12 },
  { id: "api_mobile", name: "Mobile App", keyCount: 256 },
];

type DemoNamespace = { id: string; name: string };

const DEMO_NAMESPACES: DemoNamespace[] = [
  { id: "rlns_api_requests", name: "api.requests" },
  { id: "rlns_login_attempts", name: "login.attempts" },
  { id: "rlns_signup_flows", name: "signup.flows" },
  { id: "rlns_webhook_deliveries", name: "webhook.deliveries" },
  { id: "rlns_email_sends", name: "email.sends" },
  { id: "rlns_ai_completions", name: "ai.completions" },
];

const DEMO_REQUESTS_USED = 100_585;
const DEMO_REQUESTS_QUOTA = 150_000;

type Alert = {
  id: string;
  title: string;
  meta: string;
  timestampOffsetMs: number;
  severity: "error" | "warning";
};

const DEMO_ALERTS: Alert[] = [
  {
    id: "alert-1",
    title: "5xx spike on /v1/keys.verifyKey",
    meta: "api-gateway · 142 requests in 5m",
    timestampOffsetMs: 2 * HOUR_MS,
    severity: "error",
  },
  {
    id: "alert-2",
    title: "Elevated p99 latency on auth-service",
    meta: "auth-service · 480ms (baseline 90ms)",
    timestampOffsetMs: 6 * HOUR_MS,
    severity: "warning",
  },
  {
    id: "alert-3",
    title: "Ratelimit overrides exceeded threshold",
    meta: "api.requests · 38 overrides today",
    timestampOffsetMs: 18 * HOUR_MS,
    severity: "warning",
  },
];

// Unkey Deploy per-unit pricing from unkey.com/pricing. Disk isn't published
// on the pricing page; the rate here is a stand-in so the line renders.
const VCPU_PRICE_PER_SECOND = 0.000006944;
const MEMORY_PRICE_PER_GB_SECOND = 0.000003472;
const EGRESS_PRICE_PER_GB = 0.05;
const DISK_PRICE_PER_GB_MONTH = 0.1;
const INCLUDED_CREDIT_USD = 20;

type UsageLineItem = { label: string; amount: string; price: number };

const DEMO_USAGE_LINE_ITEMS: UsageLineItem[] = [
  {
    label: "vCPU",
    amount: `${formatNumber(12_450_000)} sec`,
    price: 12_450_000 * VCPU_PRICE_PER_SECOND,
  },
  {
    label: "Memory",
    amount: `${formatNumber(8_200_000)} GB-sec`,
    price: 8_200_000 * MEMORY_PRICE_PER_GB_SECOND,
  },
  {
    label: "Egress",
    amount: `${formatNumber(412)} GB`,
    price: 412 * EGRESS_PRICE_PER_GB,
  },
  {
    label: "Disk",
    amount: `${formatNumber(24)} GB-month`,
    price: 24 * DISK_PRICE_PER_GB_MONTH,
  },
];

type ProjectCardData = {
  id: string;
  name: string;
  domain: string | null;
  commitTitle: string | null;
  commitTimestamp: number | null;
  branch: string;
  author: string | null;
  authorAvatar: string | null;
};

function toProjectCard(p: DemoProject, now: number): ProjectCardData {
  return {
    id: p.id,
    name: p.name,
    domain: p.domain,
    commitTitle: p.commitTitle,
    commitTimestamp: now - p.commitTimestampOffsetMs,
    branch: p.branch,
    author: p.author,
    authorAvatar: p.authorAvatar,
  };
}

export default function WorkspacePage() {
  const workspace = useWorkspaceNavigation();

  const now = useMemo(() => Date.now(), []);
  const projects = useMemo(() => DEMO_PROJECTS.map((p) => toProjectCard(p, now)), [now]);

  const leftColumnProjects = projects.filter((_, index) => index % 2 === 0);
  const rightColumnProjects = projects.filter((_, index) => index % 2 === 1);
  const recentProjects = useMemo(
    () => [...projects].sort((a, b) => (b.commitTimestamp ?? 0) - (a.commitTimestamp ?? 0)),
    [projects],
  );
  const usagePercent = Math.round((DEMO_REQUESTS_USED / DEMO_REQUESTS_QUOTA) * 100);

  return (
    <div className="min-h-full">
      <Navbar>
        <Navbar.Breadcrumbs icon={<Cube />}>
          <Navbar.Breadcrumbs.Link href={`/${workspace.slug}`} active>
            Overview
          </Navbar.Breadcrumbs.Link>
        </Navbar.Breadcrumbs>
      </Navbar>

      <main className="grid mx-auto max-w-7xl grid-cols-1 gap-8 p-6 lg:grid-cols-3">
        <aside className="flex flex-col gap-8">
          <section>
            <SectionHeader title="Usage" />
            <BillingOverviewCard
              workspaceSlug={workspace.slug}
              current={DEMO_REQUESTS_USED}
              quota={DEMO_REQUESTS_QUOTA}
              percent={usagePercent}
            />
          </section>

          <section>
            <SectionHeader title="Alerts" />
            <AlertsCard alerts={DEMO_ALERTS} now={now} />
          </section>

          <section>
            <SectionHeader
              title="Keyspaces"
              action={<ViewAllLink href={`/${workspace.slug}/apis`} />}
            />
            <DenseListCard emptyLabel="No keyspaces" maxVisible={VISIBLE_ROWS}>
              {DEMO_KEYSPACES.map((api) => (
                <ResourceRow
                  key={api.id}
                  href={`/${workspace.slug}/apis/${api.id}`}
                  icon={<Key className="text-gray-11" iconSize="sm-regular" />}
                  title={api.name}
                  meta={`${formatNumber(api.keyCount)} ${api.keyCount === 1 ? "key" : "keys"}`}
                />
              ))}
            </DenseListCard>
          </section>

          <section>
            <SectionHeader
              title="Ratelimit Namespaces"
              action={<ViewAllLink href={`/${workspace.slug}/ratelimits`} />}
            />
            <DenseListCard emptyLabel="No namespaces" maxVisible={VISIBLE_ROWS}>
              {DEMO_NAMESPACES.map((namespace) => (
                <ResourceRow
                  key={namespace.id}
                  href={`/${workspace.slug}/ratelimits/${namespace.id}`}
                  icon={<Gauge className="text-gray-11" iconSize="sm-regular" />}
                  title={namespace.name}
                  meta={namespace.id}
                />
              ))}
            </DenseListCard>
          </section>

          <section>
            <SectionHeader title="Recent Activity" />
            <DenseListCard emptyLabel="No recent deployments">
              {recentProjects.map((project) => (
                <RecentActivityRow
                  key={project.id}
                  project={project}
                  workspaceSlug={workspace.slug}
                />
              ))}
            </DenseListCard>
          </section>
        </aside>

        <section className="lg:col-span-2">
          <SectionHeader title="Projects" />
          <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
            <ProjectColumn projects={leftColumnProjects} workspaceSlug={workspace.slug} />
            <ProjectColumn projects={rightColumnProjects} workspaceSlug={workspace.slug} />
          </div>
        </section>
      </main>
    </div>
  );
}

function SectionHeader({ title, action }: { title: string; action?: React.ReactNode }) {
  return (
    <div className="mb-3 flex h-7 items-center justify-between">
      <h2 className="text-base font-semibold text-accent-12">{title}</h2>
      {action}
    </div>
  );
}

function ViewAllLink({ href }: { href: string }) {
  return (
    <Link
      href={href}
      className="text-xs font-medium text-gray-11 transition-colors hover:text-accent-12"
    >
      View all
    </Link>
  );
}

function ProjectColumn({
  projects,
  workspaceSlug,
}: {
  projects: ProjectCardData[];
  workspaceSlug: string;
}) {
  return (
    <div className="flex flex-col gap-4">
      {projects.map((project) => (
        <ProximityPrefetch distance={300} debounceDelay={150} key={project.id}>
          <ResourceCard
            href={`/${workspaceSlug}/projects/${project.id}`}
            name={project.name}
            domain={project.domain}
            commitTitle={project.commitTitle}
            commitTimestamp={project.commitTimestamp}
            branch={project.branch}
            author={project.author}
            authorAvatar={project.authorAvatar}
            actions={
              <ProjectActions projectId={project.id}>
                <Button
                  variant="ghost"
                  size="icon"
                  className="mb-auto shrink-0"
                  title="Project actions"
                >
                  <Dots iconSize="sm-regular" />
                </Button>
              </ProjectActions>
            }
          />
        </ProximityPrefetch>
      ))}
    </div>
  );
}

function DenseListCard({
  emptyLabel,
  maxVisible,
  children,
}: {
  emptyLabel: string;
  maxVisible?: number;
  children: React.ReactNode;
}) {
  const itemCount = Array.isArray(children) ? children.length : children ? 1 : 0;
  const scrollStyle =
    maxVisible !== undefined && itemCount > maxVisible
      ? { maxHeight: maxVisible * 52 + 8 }
      : undefined;

  return (
    <div className="overflow-hidden rounded-xl border border-gray-4 bg-grayA-1">
      {itemCount > 0 ? (
        <div className="flex flex-col gap-px overflow-y-auto p-2" style={scrollStyle}>
          {children}
        </div>
      ) : (
        <div className="px-3 py-6 text-center text-xs text-gray-10">{emptyLabel}</div>
      )}
    </div>
  );
}

function AlertsCard({ alerts, now }: { alerts: Alert[]; now: number }) {
  if (alerts.length === 0) {
    return (
      <div className="rounded-xl border border-gray-4 bg-grayA-1 px-3 py-6 text-center text-xs text-gray-10">
        No active alerts
      </div>
    );
  }
  return (
    <div className="overflow-hidden rounded-xl border border-gray-4 bg-grayA-1">
      <div className="flex flex-col gap-px p-2">
        {alerts.map((alert) => (
          <div
            key={alert.id}
            className="flex min-w-0 items-start gap-2 rounded-md px-2 py-2 transition-colors hover:bg-grayA-3"
          >
            <div className="flex size-5 shrink-0 items-center justify-center">
              <TriangleWarning2
                iconSize="sm-regular"
                className={alert.severity === "error" ? "text-error-9" : "text-warning-9"}
              />
            </div>
            <div className="min-w-0 flex-1">
              <div className="flex items-baseline justify-between gap-2">
                <p className="truncate text-sm font-medium leading-5 text-accent-12">
                  {alert.title}
                </p>
                <TimestampInfo
                  value={now - alert.timestampOffsetMs}
                  className="shrink-0 text-xs text-gray-10"
                />
              </div>
              <p className="truncate text-xs leading-4 text-gray-10">{alert.meta}</p>
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}

function ResourceRow({
  href,
  icon,
  title,
  meta,
}: {
  href: string;
  icon: React.ReactNode;
  title: string;
  meta: string;
}) {
  return (
    <Link
      href={href}
      className="flex min-w-0 items-center gap-2 rounded-md px-2 py-2 transition-colors hover:bg-grayA-3"
    >
      <div className="flex size-5 shrink-0 items-center justify-center">{icon}</div>
      <div className="min-w-0 flex-1">
        <p className="truncate text-sm font-medium leading-5 text-accent-12">{title}</p>
        <p className="truncate text-xs leading-4 text-gray-10">{meta}</p>
      </div>
    </Link>
  );
}

function RecentActivityRow({
  project,
  workspaceSlug,
}: {
  project: ProjectCardData;
  workspaceSlug: string;
}) {
  return (
    <Link
      href={`/${workspaceSlug}/projects/${project.id}`}
      className="flex min-w-0 items-start gap-2 rounded-md px-2 py-2 transition-colors hover:bg-grayA-3"
    >
      <Avatar src={project.authorAvatar} alt={project.author ?? "Unknown author"} />
      <div className="min-w-0 flex-1">
        <p className="truncate text-sm font-medium leading-5 text-accent-12">
          {project.commitTitle ?? project.name}
        </p>
        <div className="mt-0.5 flex min-w-0 items-center gap-2 text-xs text-gray-10">
          <span className="truncate font-medium text-gray-11">{project.name}</span>
          <span aria-hidden="true">·</span>
          <CodeBranch className="shrink-0 text-gray-11" iconSize="sm-regular" />
          <span className="truncate">{project.branch}</span>
          {project.commitTimestamp ? (
            <>
              <span aria-hidden="true">·</span>
              <TimestampInfo value={project.commitTimestamp} className="shrink-0" />
            </>
          ) : null}
        </div>
      </div>
    </Link>
  );
}

function formatUsd(value: number): string {
  return value.toLocaleString("en-US", {
    style: "currency",
    currency: "USD",
    minimumFractionDigits: 2,
    maximumFractionDigits: 2,
  });
}

function daysRemainingInCycle(now: Date): number {
  const lastDay = new Date(Date.UTC(now.getUTCFullYear(), now.getUTCMonth() + 1, 0)).getUTCDate();
  return Math.max(0, lastDay - now.getUTCDate());
}

function BillingOverviewCard({
  workspaceSlug,
  current,
  quota,
  percent,
}: {
  workspaceSlug: string;
  current: number;
  quota: number;
  percent: number;
}) {
  const onDemandTotal = DEMO_USAGE_LINE_ITEMS.reduce((sum, item) => sum + item.price, 0);
  const includedUsed = 0;
  const includedPercent = Math.min(100, Math.round((includedUsed / INCLUDED_CREDIT_USD) * 100));
  const daysRemaining = daysRemainingInCycle(new Date());

  return (
    <div className="overflow-hidden rounded-xl border border-gray-4 bg-grayA-1">
      <div className="flex items-center justify-between gap-3 p-4 pb-3">
        <h3 className="text-sm font-semibold text-accent-12">
          {daysRemaining} {daysRemaining === 1 ? "day" : "days"} remaining in cycle
        </h3>
        <Link
          href={`/${workspaceSlug}/settings/billing`}
          className="rounded-md border border-gray-4 px-2.5 py-1 text-xs font-medium text-accent-12 transition-colors hover:bg-grayA-3"
        >
          Billing
        </Link>
      </div>

      <div className="mx-4 rounded-lg border border-gray-4 bg-grayA-2 p-3">
        <div className="flex items-center justify-between text-xs text-gray-10">
          <span>Included Credit</span>
          <span>On-Demand Charges</span>
        </div>
        <div className="mt-1 flex items-baseline justify-between gap-3">
          <span className="text-sm font-semibold tabular-nums text-accent-12">
            {formatUsd(includedUsed)} / {formatUsd(INCLUDED_CREDIT_USD)}
          </span>
          <span className="text-sm font-semibold tabular-nums text-accent-12">
            {formatUsd(onDemandTotal)}
          </span>
        </div>
        <div className="mt-3 h-1.5 overflow-hidden rounded-full bg-gray-4">
          <div
            className="h-full rounded-full bg-accent-12 transition-[width]"
            style={{ width: `${includedPercent}%` }}
          />
        </div>
      </div>

      <div className="px-4 pt-4 pb-3 text-sm font-medium text-accent-12">
        Requests{" "}
        <span className="font-normal text-gray-10">
          {formatNumber(current)} of {formatNumber(quota)} ({percent}%)
        </span>
      </div>

      <div className="divide-y divide-gray-4 border-t border-gray-4">
        {DEMO_USAGE_LINE_ITEMS.map((item) => (
          <div
            key={item.label}
            className="flex items-center justify-between px-4 py-2.5 text-sm"
          >
            <div className="flex min-w-0 items-baseline gap-2">
              <span className="font-medium text-accent-12">{item.label}</span>
              <span className="truncate text-xs text-gray-10">{item.amount}</span>
            </div>
            <span className="shrink-0 tabular-nums text-accent-12">{formatUsd(item.price)}</span>
          </div>
        ))}
      </div>
    </div>
  );
}
