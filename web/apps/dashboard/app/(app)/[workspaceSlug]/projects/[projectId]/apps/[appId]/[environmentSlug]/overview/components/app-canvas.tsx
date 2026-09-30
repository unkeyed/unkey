"use client";

import { ago, compact } from "@/app/(app)/[workspaceSlug]/projects/_components/canvas/format";
import {
  Badge,
  GhostNode,
  Group,
  HFlow,
  Node,
  NodeHeader,
  type NodeTone,
  RowItem,
  Rows,
  StatusBadge,
} from "@/app/(app)/[workspaceSlug]/projects/_components/canvas/nodes";
import { CanvasViewport } from "@/app/(app)/[workspaceSlug]/projects/_components/canvas/viewport";
import { useOverviewWindow } from "@/app/(app)/[workspaceSlug]/projects/_components/canvas/window";
import { RectFlag } from "@/app/(app)/[workspaceSlug]/projects/_components/region/rect-flag";
import { regionInfo } from "@/app/(app)/[workspaceSlug]/projects/_components/region/region-info";
import { DEPLOYMENT_GROUP_COLOR } from "@/lib/collections/deploy/deployment-status";
import type { PolicyRow } from "@/lib/collections/deploy/policies";
import type { Policy } from "@/lib/collections/deploy/policies.schema";
import { routes } from "@/lib/navigation/routes";
import { trpc } from "@/lib/trpc/client";
import {
  Github,
  IconCodeBranchOutline18,
  IconCodeCommitOutline18,
  IconCubeOutline18,
  IconEarthOutline18,
  IconGridOutline18,
  IconLayers2Outline18,
  IconLayers3Outline18,
  IconLink4Outline18,
  IconMicrochipOutline18,
  IconNodesOutline18,
  IconShieldKeyOutline18,
  IconTerminalOutline18,
} from "@unkey/icons";
import { cn } from "@unkey/ui/src/lib/utils";
import type { Route } from "next";
import Link from "next/link";
import { type ReactNode, useMemo } from "react";
import { useAppCurrentDeployment } from "../../../hooks/use-app-current-deployment";
import { useAppEnvironment, useAppScope } from "../../environment-context";
import { usePoliciesData } from "../../policies/hooks/use-policies-data";
import { formatCpu, formatMemory } from "../../settings/components/compute/model";
import { type CardDomain, useProductionCard } from "./production-card-context";
import { STATUS_META } from "./status";

const POLICY_TYPE_SHORT: Record<Policy["type"], string> = {
  keyauth: "Key Auth",
  ratelimit: "Rate Limit",
  firewall: "Firewall",
  openapi: "OpenAPI",
  logging: "Logging",
};

function linkedKeyspaces(policies: PolicyRow[]): string[] {
  return [...new Set(policies.flatMap((p) => (p.type === "keyauth" ? p.keyauth.keyspaces : [])))];
}

type AppCanvasProps = {
  domains: CardDomain[];
  app: ReactNode;
};

export function AppCanvas({ domains, app }: AppCanvasProps) {
  const { environment } = useAppEnvironment();
  const { rowsByEnv, isLoading: policiesLoading } = usePoliciesData();
  const policies = rowsByEnv[environment.kind];
  const keyAuthIds = useMemo(() => linkedKeyspaces(policies), [policies]);

  const serviceCount = policies.length + keyAuthIds.length;

  return (
    <div className="w-full border-b bg-gray-3/40 last:rounded-b-lg last:border-b-0">
      <CanvasViewport
        label="App canvas"
        className="flex w-full min-w-[900px] items-start px-5 py-5"
      >
        <Group icon={<IconEarthOutline18 />} label={`Domains · ${domains.length}`}>
          <DomainsNode domains={domains} />
        </Group>
        <HFlow empty={domains.length === 0} />
        <Group icon={<IconCubeOutline18 />} label="App">
          {app}
        </Group>
        <HFlow empty={serviceCount === 0} />
        <Group
          icon={<IconGridOutline18 />}
          label={serviceCount > 0 ? `Services · ${serviceCount}` : "Services"}
        >
          {policiesLoading ? null : (
            <>
              <PoliciesNode policies={policies} />
              <KeyspacesNode keyAuthIds={keyAuthIds} />
            </>
          )}
        </Group>
      </CanvasViewport>
    </div>
  );
}

function DomainsNode({ domains }: { domains: CardDomain[] }) {
  const scope = useAppScope();
  const settingsHref = routes.projects.apps.settings({ ...scope, page: "domains" });
  const platform = domains.filter((d) => d.source === "platform");
  const custom = domains.filter((d) => d.source === "custom");
  return (
    <Node edge="border" link={{ href: settingsHref, label: "Domain settings" }}>
      <NodeHeader
        icon={<IconEarthOutline18 />}
        title="Domains"
        right={
          platform.length > 0 ? (
            <StatusBadge dotClass={DEPLOYMENT_GROUP_COLOR.ready}>Enabled</StatusBadge>
          ) : (
            <StatusBadge dotClass="bg-gray-7">Pending</StatusBadge>
          )
        }
      />
      <Rows>
        {platform.length > 0 ? (
          platform.map((d) => (
            <RowItem
              key={d.hostname}
              icon={<IconEarthOutline18 />}
              label="Unkey"
              value={<HostnameLink domain={d} />}
            />
          ))
        ) : (
          <RowItem icon={<IconEarthOutline18 />} label="Unkey" value="Assigned on first deploy" />
        )}
        {custom.length > 0 ? (
          custom.map((d) => (
            <RowItem
              key={d.hostname}
              icon={<IconLink4Outline18 />}
              label="Custom"
              value={<HostnameLink domain={d} />}
            />
          ))
        ) : (
          <RowItem
            icon={<IconLink4Outline18 />}
            label="Custom"
            value={
              <Link
                href={settingsHref}
                className="relative z-10 font-sans text-gray-10 hover:text-gray-12"
              >
                Add domain
              </Link>
            }
          />
        )}
      </Rows>
    </Node>
  );
}

function HostnameLink({ domain }: { domain: CardDomain }) {
  return (
    <a
      href={domain.url}
      target="_blank"
      rel="noopener noreferrer"
      className="relative z-10 hover:underline"
    >
      {domain.hostname}
    </a>
  );
}

export function appIcon(app: ReturnType<typeof useAppCurrentDeployment>["app"]): ReactNode {
  if (app?.sourceType === "oci") {
    return <IconLayers2Outline18 />;
  }
  return app?.repositoryFullName ? <Github /> : <IconTerminalOutline18 />;
}

export function AppNode() {
  const { deployment, status, isRolledBack, deploymentHref } = useProductionCard();
  const { app } = useAppCurrentDeployment();
  const tone: NodeTone = status === "failed" || status === "crashing" ? "error" : "default";

  const instances = deployment.instances ?? [];
  const running = instances.filter((i) => i.status === "running").length;
  const regions =
    instances.length > 0
      ? [...new Set(instances.map((i) => i.region.name))]
      : deployment.desiredRegions.map((r) => r.region.name);
  const image = deployment.requestedImage ?? deployment.resolvedImage;

  return (
    <Node
      edge="border"
      tone={tone}
      link={{ href: deploymentHref, label: `${app?.name ?? "App"} deployment` }}
    >
      <NodeHeader
        icon={appIcon(app)}
        title={app?.name ?? "App"}
        right={
          <StatusBadge dotClass={STATUS_META[status].dotClass} tone={tone}>
            {STATUS_META[status].label}
            {isRolledBack && <span className="text-warning-11">· Rolled back</span>}
          </StatusBadge>
        }
      />
      <Rows>
        {deployment.source === "git" ? (
          <>
            <RowItem
              icon={<IconCodeBranchOutline18 />}
              label="Branch"
              value={deployment.gitBranch ?? "main"}
              hint={deployment.gitCommitSha?.slice(0, 7)}
            />
            <RowItem
              icon={<IconCodeCommitOutline18 />}
              label="Commit"
              value={
                <span className="font-sans">
                  {deployment.gitCommitMessage?.split("\n")[0] ?? "—"}
                </span>
              }
              hint={ago(deployment.createdAt)}
            />
          </>
        ) : (
          <RowItem
            icon={<IconLayers2Outline18 />}
            label="Image"
            value={image ?? "Unknown"}
            hint={ago(deployment.createdAt)}
          />
        )}
      </Rows>
      <Rows>
        <RowItem
          icon={<IconEarthOutline18 />}
          label="Regions"
          value={
            regions.length > 0 ? (
              <span className="flex items-center justify-end gap-2">
                {regions.map((name) => (
                  <span key={name} className="flex items-center gap-1">
                    <RectFlag flag={regionInfo(name).flag} size="sm" />
                    {name}
                  </span>
                ))}
              </span>
            ) : (
              "—"
            )
          }
        />
        <RowItem icon={<IconLayers3Outline18 />} label="Instances" value={running} hint="running" />
        <RowItem
          icon={<IconMicrochipOutline18 />}
          label="Resources"
          value={formatCpu(deployment.cpuMillicores)}
          hint={`· ${formatMemory(deployment.memoryMib)}`}
        />
      </Rows>
    </Node>
  );
}

function PoliciesNode({ policies }: { policies: PolicyRow[] }) {
  const scope = useAppScope();
  const href = routes.projects.apps.policies(scope);
  if (policies.length === 0) {
    return (
      <GhostNode
        icon={<IconShieldKeyOutline18 />}
        title="Add a policy"
        description="Auth, ratelimits and firewall rules at the gateway."
        href={href}
      />
    );
  }
  return (
    <Node edge="border" link={{ href, label: "Policies" }}>
      <NodeHeader
        icon={<IconShieldKeyOutline18 />}
        title="Policies"
        right={<Badge>{policies.length}</Badge>}
      />
      <Rows>
        {policies.map((p) => (
          <span key={p.id} className="flex min-w-0 items-center gap-2">
            <span
              className={cn(
                "size-1.5 shrink-0 rounded-full",
                p.enabled ? "bg-success-9" : "bg-gray-7",
              )}
            />
            <span className="truncate text-gray-12">{p.name}</span>
            <span className="ml-auto shrink-0 font-mono text-[10px] text-gray-10">
              {POLICY_TYPE_SHORT[p.type]}
            </span>
          </span>
        ))}
      </Rows>
    </Node>
  );
}

function KeyspacesNode({ keyAuthIds }: { keyAuthIds: string[] }) {
  const scope = useAppScope();
  const keyspaces = trpc.deploy.project.keyspaces.useQuery({ projectId: scope.projectId });
  const available = trpc.deploy.environmentSettings.getAvailableKeyspaces.useQuery();
  if (keyAuthIds.length === 0) {
    return (
      <GhostNode
        icon={<IconNodesOutline18 />}
        title="Protect it with keys"
        description="Add a Key Auth policy to verify API keys."
        href={routes.projects.apps.policies(scope)}
      />
    );
  }
  const byKeyAuth = new Map(keyspaces.data?.map((k) => [k.keyAuthId, k]));
  return (
    <Node edge="border" link={{ href: routes.apis.list(scope), label: "Keyspaces" }}>
      <NodeHeader
        icon={<IconNodesOutline18 />}
        title="Keyspaces"
        right={<Badge>{keyAuthIds.length}</Badge>}
      />
      <Rows>
        {keyAuthIds.map((id) => {
          const ks = byKeyAuth.get(id);
          return (
            <KeyspaceRow
              key={id}
              keyAuthId={id}
              name={ks?.name ?? available.data?.[id]?.api.name ?? id}
              keyCount={ks?.keyCount}
              href={ks ? routes.apis.detail({ ...scope, apiId: ks.apiId }) : undefined}
            />
          );
        })}
      </Rows>
    </Node>
  );
}

function KeyspaceRow({
  keyAuthId,
  name,
  keyCount,
  href,
}: {
  keyAuthId: string;
  name: string;
  keyCount: number | undefined;
  href: Route | undefined;
}) {
  const window = useOverviewWindow();
  const { data } = trpc.api.overview.timeseries.useQuery(
    { keyspaceId: keyAuthId, ...window, since: "" },
    { trpc: { context: { skipBatch: true } } },
  );
  const total = data?.timeseries?.reduce((a, p) => a + p.y.total, 0);
  const content = (
    <>
      <span className="truncate text-gray-12">{name}</span>
      <span className="ml-auto shrink-0 font-mono text-[10px] text-gray-10">
        {keyCount == null ? "—" : compact(keyCount)} keys · {total == null ? "…" : compact(total)}{" "}
        7d
      </span>
    </>
  );
  return href ? (
    <Link href={href} className="relative z-10 flex min-w-0 items-center gap-2 hover:underline">
      {content}
    </Link>
  ) : (
    <span className="flex min-w-0 items-center gap-2">{content}</span>
  );
}
