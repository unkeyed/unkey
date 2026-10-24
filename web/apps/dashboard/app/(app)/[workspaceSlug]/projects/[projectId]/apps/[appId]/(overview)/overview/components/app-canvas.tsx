"use client";

import { SourceIcon } from "@/components/deploy/app-source";
import { RegionFlag } from "@/components/region-flag";
import {
  DEPLOYMENT_STATUS_LABELS,
  statusGroupOf,
} from "@/lib/collections/deploy/deployment-status";
import type { Deployment } from "@/lib/collections/deploy/deployments";
import {
  type InstanceSummary,
  groupInstancesByRegion,
  summarizeInstances,
} from "@/lib/collections/deploy/instance-status";
import type { PolicyRow } from "@/lib/collections/deploy/policies";
import { formatCpu, formatMemory } from "@/lib/utils/deployment-formatters";
import {
  IconCodeBranchOutline18,
  IconCodeCommitOutline18,
  IconCubeOutline18,
  IconEarthOutline18,
  IconGridOutline18,
  IconLayers2Outline18,
  IconLayers3Outline18,
  IconLink4Outline18,
  IconMicrochipOutline18,
  IconShieldKeyOutline18,
  IconTerminalOutline18,
  IconTriangleWarningOutline18,
} from "@unkey/icons";
import { match } from "@unkey/match";
import { Badge, Button, InfoTooltip, Skeleton, useElapsed } from "@unkey/ui";
import { cn } from "cn";
import type { Route } from "next";
import Link from "next/link";
import type { ReactNode } from "react";
import { DeploymentStatusIndicator, StatusDot } from "../../../components/deployment-status-dot";
import type { DisplayDomain } from "../../../components/display-domain";
import { useProductionPolicies } from "../../policies/hooks/use-policies-data";
import { POLICY_KINDS } from "../../policies/policy-kinds";
import {
  Canvas,
  GhostNode,
  Group,
  HFlow,
  MoreRow,
  Node,
  NodeHeader,
  type NodeTone,
  RowItem,
  Rows,
  StatusBadge,
} from "./canvas/nodes";
import { useProductionCard } from "./production-card-context";
import { type DeploymentDisplayStatus, STATUS_META } from "./status";

const NODE_ROW_LIMIT = 5;

export function AppCanvas() {
  const { isCurrent, canvas } = useProductionCard();
  const { domains, domainsHref, policiesHref } = canvas;
  const { policies, view, retry, isRetrying } = useProductionPolicies();
  const listed = view.type === "list";

  return (
    <Canvas label="App canvas">
      <Group
        icon={<IconEarthOutline18 />}
        label={domains.length > 0 ? `Domains · ${domains.length}` : "Domains"}
      >
        <DomainsNode domains={domains} href={domainsHref} isCurrent={isCurrent} />
      </Group>
      <HFlow empty={domains.length === 0} />
      <Group icon={<IconCubeOutline18 />} label="App">
        <AppNode />
      </Group>
      <HFlow empty={!listed} />
      <Group
        icon={<IconGridOutline18 />}
        label={listed ? `Services · ${policies.length}` : "Services"}
      >
        {match(view.type)
          .with("loading", () => <PoliciesLoadingNode />)
          .with("error", () => <PoliciesErrorNode onRetry={retry} retrying={isRetrying} />)
          .with("empty", () => (
            <GhostNode
              icon={<IconShieldKeyOutline18 />}
              name="Add a policy"
              description="Auth, ratelimits and firewall rules at the gateway."
              href={policiesHref}
            />
          ))
          .with("list", () => <PoliciesNode policies={policies} href={policiesHref} />)
          .exhaustive()}
      </Group>
    </Canvas>
  );
}

const DOMAIN_ROW: Record<DisplayDomain["source"], { icon: ReactNode; label: string }> = {
  platform: { icon: <IconEarthOutline18 />, label: "Unkey" },
  custom: { icon: <IconLink4Outline18 />, label: "Custom" },
};

function DomainsNode({
  domains,
  href,
  isCurrent,
}: {
  domains: ReadonlyArray<DisplayDomain>;
  href: Route;
  isCurrent: boolean;
}) {
  const hasPlatform = domains.some((d) => d.source === "platform");
  const hasCustom = domains.some((d) => d.source === "custom");
  const shown = domains.slice(0, hasPlatform ? NODE_ROW_LIMIT : NODE_ROW_LIMIT - 1);
  return (
    <Node link={{ href, label: "Domain settings" }}>
      <NodeHeader icon={<IconEarthOutline18 />} name="Domains" />
      <Rows>
        {!hasPlatform && (
          <RowItem
            icon={DOMAIN_ROW.platform.icon}
            label={DOMAIN_ROW.platform.label}
            value="Not assigned yet"
          />
        )}
        {shown.map((d) => (
          <RowItem
            key={d.id}
            icon={DOMAIN_ROW[d.source].icon}
            label={DOMAIN_ROW[d.source].label}
            value={<HostnameLink domain={d} />}
          />
        ))}
        {isCurrent && !hasCustom && (
          <RowItem
            icon={DOMAIN_ROW.custom.icon}
            label={DOMAIN_ROW.custom.label}
            value={
              <Link
                href={href}
                className="relative z-10 font-sans text-gray-10 hover:text-gray-12 rounded-sm focus-visible:outline-hidden focus-visible:ring-2 focus-visible:ring-gray-7"
              >
                Add domain
              </Link>
            }
          />
        )}
        <MoreRow count={domains.length - shown.length} href={href} />
      </Rows>
    </Node>
  );
}

function HostnameLink({ domain }: { domain: DisplayDomain }) {
  return (
    <a
      href={domain.url}
      target="_blank"
      rel="noopener noreferrer"
      className="relative z-10 hover:underline rounded-sm focus-visible:outline-hidden focus-visible:ring-2 focus-visible:ring-gray-7"
    >
      {domain.hostname}
    </a>
  );
}

function AppNode() {
  const { deployment, status, isCurrent, isRolledBack, deploymentHref, canvas } =
    useProductionCard();
  const badge = appStatusOf({ deployment, status, isCurrent });
  const deployedAgo = useElapsed(deployment.createdAt, "narrow");
  const summary = summarizeInstances(deployment);
  const regions = groupInstancesByRegion(deployment).map((r) => r.name);
  const image = deployment.requestedImage ?? deployment.resolvedImage;
  const commitMessage = deployment.gitCommitMessage?.split("\n")[0];

  return (
    <Node tone={badge.tone} link={{ href: deploymentHref, label: `${canvas.appName} deployment` }}>
      <NodeHeader
        icon={<SourceIcon source={canvas.appSource} />}
        name={canvas.appName}
        right={
          <StatusBadge indicator={badge.indicator} tone={badge.tone}>
            {badge.label}
            {isRolledBack && <span className="text-warning-11">· Rolled back</span>}
          </StatusBadge>
        }
      />
      <Rows>
        {match(deployment.source)
          .with("git", () => (
            <>
              <RowItem
                icon={<IconCodeBranchOutline18 />}
                label="Branch"
                value={deployment.gitBranch || "—"}
                fullValue={deployment.gitBranch || undefined}
                hint={deployment.gitCommitSha?.slice(0, 7)}
                copy={deployment.gitCommitSha ?? undefined}
              />
              <RowItem
                icon={<IconCodeCommitOutline18 />}
                label="Commit"
                value={<span className="font-sans">{commitMessage || "—"}</span>}
                fullValue={commitMessage || undefined}
                hint={deployedAgo}
              />
            </>
          ))
          .with("oci", () => (
            <RowItem
              icon={<IconLayers2Outline18 />}
              label="Image"
              value={image ?? "Unknown"}
              fullValue={image ?? undefined}
              hint={deployedAgo}
              copy={image ?? undefined}
            />
          ))
          .with("unknown", () => (
            <>
              <RowItem
                icon={<IconTerminalOutline18 />}
                label="Source"
                value="Unknown"
                hint={deployedAgo}
              />
              {canvas.defaultBranch && (
                <RowItem
                  icon={<IconCodeBranchOutline18 />}
                  label="Default branch"
                  value={canvas.defaultBranch}
                />
              )}
            </>
          ))
          .exhaustive()}
      </Rows>
      <Rows>
        <RowItem
          icon={<IconEarthOutline18 />}
          label="Regions"
          value={
            regions.length > 0 ? (
              <InfoTooltip asChild content={regions.join(", ")} position={{ side: "top" }}>
                <span
                  // biome-ignore lint/a11y/noNoninteractiveTabindex: focusable so keyboard users can open the region names tooltip
                  tabIndex={0}
                  aria-label={regions.join(", ")}
                  className="relative z-10 flex items-center gap-1 rounded-sm focus-visible:outline-hidden focus-visible:ring-2 focus-visible:ring-gray-7"
                >
                  {regions.map((name) => (
                    <RegionFlag key={name} region={name} size="xs" shape="rect" />
                  ))}
                </span>
              </InfoTooltip>
            ) : (
              "—"
            )
          }
        />
        <RowItem
          icon={<IconLayers3Outline18 />}
          label="Instances"
          value={instancesValue(summary)}
        />
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

function PoliciesNode({ policies, href }: { policies: PolicyRow[]; href: Route }) {
  return (
    <Node link={{ href, label: "Policies" }}>
      <NodeHeader
        icon={<IconShieldKeyOutline18 />}
        name="Policies"
        right={<Badge variant="count">{policies.length}</Badge>}
      />
      <Rows>
        {policies.slice(0, NODE_ROW_LIMIT).map((p) => (
          <span key={p.id} className="flex min-w-0 items-center gap-2">
            <span className={cn("truncate", p.enabled ? "text-gray-12" : "text-gray-10")}>
              {p.name}
              {!p.enabled && <span className="sr-only"> (disabled)</span>}
            </span>
            <span className="ml-auto shrink-0 font-mono text-3xs text-gray-10">
              {POLICY_KINDS[p.type].label}
            </span>
          </span>
        ))}
        <MoreRow count={policies.length - NODE_ROW_LIMIT} href={href} />
      </Rows>
    </Node>
  );
}

function PoliciesErrorNode({ onRetry, retrying }: { onRetry: () => void; retrying: boolean }) {
  return (
    <Node>
      <NodeHeader icon={<IconShieldKeyOutline18 />} name="Policies" />
      <Rows>
        <span className="flex items-center gap-2">
          <IconTriangleWarningOutline18 className="size-3 shrink-0 text-error-9" />
          <span className="text-gray-11">Couldn't load</span>
          <Button
            variant="ghost"
            size="sm"
            className="ml-auto"
            onClick={onRetry}
            loading={retrying}
          >
            Retry
          </Button>
        </span>
      </Rows>
    </Node>
  );
}

function PoliciesLoadingNode() {
  return <Skeleton className="h-28 w-full rounded-lg" />;
}

type AppStatus = { label: string; indicator: ReactNode; tone: NodeTone };

function appStatusOf({
  deployment,
  status,
  isCurrent,
}: {
  deployment: Deployment;
  status: DeploymentDisplayStatus;
  isCurrent: boolean;
}): AppStatus {
  const meta = STATUS_META[status];
  if (isCurrent) {
    return {
      label: meta.label,
      indicator: <StatusDot colorClass={meta.dotClass} size="sm" />,
      tone: meta.tone,
    };
  }
  return {
    label: DEPLOYMENT_STATUS_LABELS[deployment.status],
    indicator: <DeploymentStatusIndicator status={deployment.status} size="sm" />,
    tone: statusGroupOf(deployment.status) === "failed" ? "error" : "default",
  };
}

function instancesValue(summary: InstanceSummary | null): string {
  if (!summary) {
    return "—";
  }
  return summary.running === summary.total
    ? `${summary.running}`
    : `${summary.running}/${summary.total}`;
}
