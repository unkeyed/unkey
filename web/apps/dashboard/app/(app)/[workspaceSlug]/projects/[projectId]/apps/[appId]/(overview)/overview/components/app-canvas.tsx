"use client";

import { SourceIcon } from "@/components/deploy/app-source";
import { RegionFlag } from "@/components/region-flag";
import {
  DEPLOYMENT_STATUS_LABELS,
  statusGroupOf,
} from "@/lib/collections/deploy/deployment-status";
import type { Deployment, DeploymentSummary } from "@/lib/collections/deploy/deployments";
import {
  type InstanceSummary,
  groupInstancesByRegion,
  summarizeInstances,
} from "@/lib/collections/deploy/instance-status";
import type { PolicyRow } from "@/lib/collections/deploy/policies";
import { imageRefTag } from "@/lib/docker-image-ref";
import { formatCpu, formatMemory } from "@/lib/utils/deployment-formatters";
import {
  IconClockOutline18,
  IconCodeBranchOutline18,
  IconCubeOutline18,
  IconEarthOutline18,
  IconEarthPlusOutline18,
  IconGridOutline18,
  IconLayers2Outline18,
  IconLink4Outline18,
  IconMicrochipOutline18,
  IconShieldKeyOutline18,
  IconTerminalOutline18,
  IconTriangleWarningOutline18,
} from "@unkey/icons";
import { match } from "@unkey/match";
import { Badge, Button, InfoTooltip, Skeleton, TimestampInfo } from "@unkey/ui";
import { cn } from "cn";
import type { Route } from "next";
import type { ReactNode } from "react";
import { DeploymentStatusIndicator, StatusDot } from "../../../components/deployment-status-dot";
import type { DisplayDomain } from "../../../components/display-domain";
import { Avatar } from "../../../components/git-avatar";
import { useProductionPolicies } from "../../policies/hooks/use-policies-data";
import { POLICY_KINDS } from "../../policies/policy-kinds";
import {
  Canvas,
  GhostNode,
  Group,
  HFlow,
  IconRow,
  MoreRow,
  Node,
  NodeFooter,
  NodeHeader,
  type NodeTone,
  RowItem,
  Rows,
} from "./canvas/nodes";
import type { DeployStamp } from "./deploy-stamp";
import { useProductionCard } from "./production-card-context";
import {
  type DeploymentDisplayStatus,
  GROUP_BADGE_VARIANT,
  STATUS_META,
  type StatusBadgeVariant,
} from "./status";

const NODE_ROW_LIMIT = 3;

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
  const shown = domains.slice(0, NODE_ROW_LIMIT - Number(!hasPlatform));
  return (
    <Node
      link={{ href, label: "Domain settings" }}
      footer={
        addHref && (
          <NodeFooter href={addHref} icon={<IconEarthPlusOutline18 />}>
            Add a custom domain
          </NodeFooter>
        )
      }
    >
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
        className="relative z-10 flex min-w-0 text-gray-12 hover:underline rounded-sm focus-visible:outline-hidden focus-visible:ring-2 focus-visible:ring-gray-7"
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
  const { deployment, status, isCurrent, stamp, replaced, deploymentHref, canvas } =
    useProductionCard();
  const badge = appStatusOf({ deployment, status, isCurrent });
  const summary = summarizeInstances(deployment);
  const regions = groupInstancesByRegion(deployment).map((r) => r.name);
  const image = imageRefOf(deployment)?.note ?? null;
  const commit = commitRefOf(deployment);
  const replacedRef = replaced && REF_OF_SOURCE[replaced.source](replaced);
  const replacedChip = replacedRef ? <RefChip value={replacedRef} replaced /> : undefined;

  return (
    <Node tone={badge.tone} link={{ href: deploymentHref, label: `${canvas.appName} deployment` }}>
      <NodeHeader
        icon={<SourceIcon source={canvas.appSource} />}
        name={canvas.appName}
        right={
          <Badge variant={badge.variant} size="compact" className="-my-0.5 -mr-1.5">
            {badge.indicator}
            {badge.label}
          </Badge>
        }
      />
      <Rows>
        {match(deployment.source)
          .with("git", () => (
            <RowItem
              icon={<IconCodeBranchOutline18 />}
              label="Branch"
              value={deployment.gitBranch || "—"}
              fullValue={deployment.gitBranch || undefined}
              mono
              hint={commit ? <RefChip value={commit} /> : undefined}
              detail={replacedChip}
            />
          ))
          .with("oci", () => (
            <RowItem
              icon={<IconLayers2Outline18 />}
              label="Image"
              value={image ?? "Unknown"}
              fullValue={image ?? undefined}
              copy={image ?? undefined}
              detail={replacedChip}
            />
          ))
          .with("unknown", () => (
            <>
              <RowItem icon={<IconTerminalOutline18 />} label="Source" value="Unknown" />
              {canvas.defaultBranch && (
                <RowItem
                  icon={<IconCodeBranchOutline18 />}
                  label="Default branch"
                  value={canvas.defaultBranch}
                  mono
                />
              )}
            </>
          ))
          .exhaustive()}
        {stamp && <DeployedRow stamp={stamp} />}
        <RowItem
          icon={<IconEarthOutline18 />}
          label="Regions"
          value={<RegionFlags regions={regions} />}
          hint={summary ? `${instancesValue(summary)} running` : undefined}
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

type Ref = { shown: string; mono: boolean; code: string; note: string | null };

function commitRefOf(deployment: DeploymentSummary): Ref | null {
  if (!deployment.gitCommitSha) {
    return null;
  }
  const code = deployment.gitCommitSha.slice(0, 7);
  const note = deployment.gitCommitMessage || null;
  return { shown: note ?? code, mono: note === null, code, note };
}

function imageRefOf(deployment: DeploymentSummary): Ref | null {
  const image = deployment.requestedImage ?? deployment.resolvedImage;
  if (!image) {
    return null;
  }
  const code = imageRefTag(image);
  return { shown: code, mono: true, code, note: image };
}

const REF_OF_SOURCE: Record<Deployment["source"], (deployment: DeploymentSummary) => Ref | null> = {
  git: commitRefOf,
  oci: imageRefOf,
  unknown: () => null,
};

function RefChip({ value, replaced = false }: { value: Ref; replaced?: boolean }) {
  const { shown, mono, code, note } = value;
  return (
    <InfoTooltip
      asChild
      position={{ side: "top" }}
      content={
        <span data-sentry-mask className="flex max-w-80 flex-col gap-0.5">
          {replaced && <span className="opacity-70">Replaced</span>}
          <span className={cn("flex min-w-0 gap-1.5", replaced && "line-through")}>
            <span className="shrink-0 font-mono">{code}</span>
            {note && <span className="truncate">{note}</span>}
          </span>
        </span>
      }
    >
      <button
        type="button"
        data-sentry-mask
        aria-label={[replaced && "Replaced", code, note].filter(Boolean).join(" ")}
        className={cn(
          "relative z-10 block min-w-0 max-w-full cursor-default truncate rounded-sm text-left focus-visible:outline-hidden focus-visible:ring-2 focus-visible:ring-gray-7",
          mono && "font-mono",
          replaced && "line-through",
        )}
      >
        {shown}
      </button>
    </InfoTooltip>
  );
}

function DeployedRow({ stamp }: { stamp: DeployStamp }) {
  const { actor } = stamp;
  return (
    <RowItem
      icon={<IconClockOutline18 />}
      label={stamp.verb}
      value={
        <span className="flex min-w-0 items-center gap-1.5">
          <TimestampInfo
            value={stamp.at}
            displayType="relative"
            className="relative z-10 shrink-0 rounded-sm focus-visible:outline-hidden focus-visible:ring-2 focus-visible:ring-gray-7"
          />
          {actor?.avatarUrl && (
            <InfoTooltip
              asChild
              content={
                <span data-sentry-mask>
                  {stamp.verb} by {actor.name}
                </span>
              }
              position={{ side: "top" }}
            >
              <button
                type="button"
                data-sentry-mask
                aria-label={`${stamp.verb} by ${actor.name}`}
                className="relative z-10 flex shrink-0 cursor-default rounded-full focus-visible:outline-hidden focus-visible:ring-2 focus-visible:ring-gray-7"
              >
                <Avatar src={actor.avatarUrl} alt="" className="size-4" />
              </button>
            </InfoTooltip>
          )}
        </span>
      }
    />
  );
}

function RegionFlags({ regions }: { regions: string[] }) {
  if (regions.length === 0) {
    return <>—</>;
  }
  const names = regions.join(", ");
  return (
    <InfoTooltip asChild content={names} position={{ side: "top" }}>
      <button
        type="button"
        aria-label={names}
        className="relative z-10 flex cursor-default items-center gap-1 rounded-sm focus-visible:outline-hidden focus-visible:ring-2 focus-visible:ring-gray-7"
      >
        {regions.map((name) => (
          <RegionFlag key={name} region={name} size="xs" shape="rect" />
        ))}
      </button>
    </InfoTooltip>
  );
}

function PoliciesNode({ policies, href }: { policies: PolicyRow[]; href: Route }) {
  return (
    <Node link={{ href, label: "Policies" }}>
      <NodeHeader icon={<IconShieldKeyOutline18 />} name="Policies" />
      <Rows>
        {policies.slice(0, NODE_ROW_LIMIT).map((p) => {
          const kind = POLICY_KINDS[p.type];
          return (
            <IconRow key={p.id} icon={<kind.Icon />}>
              <span className={cn("truncate", p.enabled ? "text-gray-12" : "text-gray-10")}>
                {p.name}
                {!p.enabled && <span className="sr-only"> (disabled)</span>}
              </span>
              <span className="ml-auto shrink-0 text-gray-10">{kind.label}</span>
            </IconRow>
          );
        })}
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
  return <Skeleton className="h-31 w-full rounded-lg" />;
}

type AppStatus = {
  label: string;
  indicator: ReactNode;
  tone: NodeTone;
  variant: StatusBadgeVariant;
};

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
  const shown =
    isCurrent || status === "crashing"
      ? {
          label: meta.label,
          indicator: <StatusDot colorClass={meta.dotClass} size="sm" />,
          variant: meta.badge,
        }
      : {
          label: DEPLOYMENT_STATUS_LABELS[deployment.status],
          indicator: <DeploymentStatusIndicator status={deployment.status} size="sm" />,
          variant: GROUP_BADGE_VARIANT[statusGroupOf(deployment.status)],
        };
  return { ...shown, tone: shown.variant === "error" ? "error" : "default" };
}

function instancesValue(summary: InstanceSummary): string {
  return summary.running === summary.total
    ? `${summary.running}`
    : `${summary.running}/${summary.total}`;
}
