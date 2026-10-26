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
  IconPlusOutline12,
  IconShieldKeyOutline18,
  IconTerminalOutline18,
  IconTriangleWarningOutline18,
} from "@unkey/icons";
import { match } from "@unkey/match";
import { Button, InfoTooltip, Skeleton, useElapsed } from "@unkey/ui";
import { cn } from "cn";
import type { Route } from "next";
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
  IconRow,
  LinkRow,
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
  const { canvas } = useProductionCard();
  const { domains, domainsHref, addDomainHref, policiesHref, addPolicyHref } = canvas;
  const { policies, view, retry, isRetrying } = useProductionPolicies();
  const listed = view.type === "list";

  return (
    <Canvas label="App canvas">
      <Group
        icon={<IconEarthOutline18 />}
        label={domains.length > 0 ? `Domains · ${domains.length}` : "Domains"}
      >
        <DomainsNode domains={domains} href={domainsHref} addHref={addDomainHref} />
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
              href={addPolicyHref}
            />
          ))
          .with("list", () => <PoliciesNode policies={policies} href={policiesHref} />)
          .exhaustive()}
      </Group>
    </Canvas>
  );
}

const DOMAIN_ICON: Record<DisplayDomain["source"], ReactNode> = {
  platform: <IconEarthOutline18 />,
  custom: <IconLink4Outline18 />,
};

function DomainsNode({
  domains,
  href,
  addHref,
}: {
  domains: ReadonlyArray<DisplayDomain>;
  href: Route;
  addHref: Route | null;
}) {
  const hasPlatform = domains.some((d) => d.source === "platform");
  const reserved = Number(!hasPlatform) + Number(addHref !== null);
  const shown = domains.slice(0, NODE_ROW_LIMIT - reserved);
  return (
    <Node link={{ href, label: "Domain settings" }}>
      <NodeHeader icon={<IconEarthOutline18 />} name="Domains" />
      <Rows>
        {!hasPlatform && (
          <IconRow icon={DOMAIN_ICON.platform}>
            <span className="text-gray-10">Unkey domain not assigned yet</span>
          </IconRow>
        )}
        {shown.map((d) => (
          <IconRow key={d.id} icon={DOMAIN_ICON[d.source]}>
            <HostnameLink domain={d} />
          </IconRow>
        ))}
        <MoreRow count={domains.length - shown.length} href={href} />
        {addHref && (
          <LinkRow
            href={addHref}
            icon={<IconPlusOutline12 className="size-2.5" />}
            className="mt-1"
          >
            Add domain
          </LinkRow>
        )}
      </Rows>
    </Node>
  );
}

function HostnameLink({ domain }: { domain: DisplayDomain }) {
  const { head, apex } = splitApex(domain.hostname);
  return (
    <InfoTooltip asChild content={domain.hostname} position={{ side: "top" }}>
      <a
        href={domain.url}
        target="_blank"
        rel="noopener noreferrer"
        className="relative z-10 flex min-w-0 font-mono text-gray-12 hover:underline rounded-sm focus-visible:outline-hidden focus-visible:ring-2 focus-visible:ring-gray-7"
      >
        <span className="truncate">{head}</span>
        <span className="max-w-full shrink-0 truncate">{apex}</span>
      </a>
    </InfoTooltip>
  );
}

function splitApex(hostname: string): { head: string; apex: string } {
  const labels = hostname.split(".");
  const apex = labels.slice(-2).join(".");
  return { head: hostname.slice(0, hostname.length - apex.length), apex };
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
              value={<span className="font-sans">{image ?? "Unknown"}</span>}
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
        right={<span className="text-2xs tabular-nums text-gray-10">{policies.length}</span>}
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
  if (isCurrent || status === "crashing") {
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
