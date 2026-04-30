"use client";

import { trpc } from "@/lib/trpc/client";
import {
  Ban,
  ChevronLeft,
  CircleCheck,
  Database,
  HalfDottedCirclePlay,
  Trash,
  TriangleWarning,
} from "@unkey/icons";
import {
  Badge,
  Button,
  Empty,
  SettingCard,
  SettingsDangerZone,
  TimestampInfo,
  toast,
} from "@unkey/ui";
import Link from "next/link";
import { useRouter } from "next/navigation";

type Props = {
  scope: "project" | "workspace";
  workspaceSlug: string;
  projectId?: string;
  drainId: string;
};

export function LogDrainDetail({ scope, workspaceSlug, projectId, drainId }: Props) {
  const router = useRouter();
  const utils = trpc.useUtils();
  const list = trpc.deploy.logDrain.list.useQuery({ scope, projectId });
  const drain = list.data?.find((d) => d.id === drainId);

  const baseHref =
    scope === "project"
      ? `/${workspaceSlug}/projects/${projectId}/log-drains`
      : `/${workspaceSlug}/settings/log-drains`;

  const pause = trpc.deploy.logDrain.pause.useMutation({
    onSuccess: () => {
      toast.success("Drain paused");
      void utils.deploy.logDrain.list.invalidate();
    },
    onError: (err) => toast.error(err.message),
  });
  const resume = trpc.deploy.logDrain.resume.useMutation({
    onSuccess: () => {
      toast.success("Drain resumed");
      void utils.deploy.logDrain.list.invalidate();
    },
    onError: (err) => toast.error(err.message),
  });
  const remove = trpc.deploy.logDrain.delete.useMutation({
    onSuccess: () => {
      toast.success("Drain deleted");
      router.push(baseHref);
    },
    onError: (err) => toast.error(err.message),
  });

  if (list.isLoading) {
    return <DetailSkeleton />;
  }

  if (!drain) {
    return (
      <div className="border border-grayA-4 rounded-[14px] overflow-hidden mt-8 mx-auto max-w-3xl">
        <div className="w-full flex justify-center items-center py-16 px-4">
          <Empty className="w-[400px] flex items-start">
            <Empty.Title>Drain not found</Empty.Title>
            <Empty.Description className="text-left">
              It may have been deleted in another tab.
            </Empty.Description>
            <Empty.Actions className="mt-4 justify-start">
              <Link href={baseHref}>
                <Button size="md" variant="outline">
                  Back to log drains
                </Button>
              </Link>
            </Empty.Actions>
          </Empty>
        </div>
      </div>
    );
  }

  return (
    <div className="mx-auto max-w-4xl px-4 py-8 flex flex-col gap-6 pb-14">
      {/* Header */}
      <div className="flex flex-col gap-4">
        <Link
          href={baseHref}
          className="flex items-center gap-1.5 text-sm text-gray-11 hover:text-gray-12 transition-colors w-fit"
        >
          <ChevronLeft className="size-3.5" />
          Back to log drains
        </Link>
        <div className="flex items-start justify-between gap-4">
          <div className="flex items-start gap-3 min-w-0">
            <div className="flex items-center justify-center size-10 rounded-lg bg-grayA-3 shrink-0">
              <ProviderIcon provider={drain.provider} className="size-5 text-gray-11" />
            </div>
            <div className="flex flex-col gap-1 min-w-0">
              <h1 className="font-semibold text-gray-12 text-2xl truncate">{drain.name}</h1>
              <div className="flex items-center gap-2">
                <span className="text-sm text-gray-11 capitalize">{drain.provider}</span>
                <span className="text-gray-8">•</span>
                <Status drain={drain} />
              </div>
            </div>
          </div>
          <div className="flex items-center gap-2 shrink-0">
            {drain.enabled ? (
              <Button
                variant="outline"
                size="md"
                onClick={() => pause.mutate({ id: drain.id })}
                loading={pause.isLoading}
              >
                <Ban className="size-4" />
                Pause
              </Button>
            ) : (
              <Button
                variant="primary"
                size="md"
                onClick={() => resume.mutate({ id: drain.id })}
                loading={resume.isLoading}
              >
                <HalfDottedCirclePlay className="size-4" />
                Resume
              </Button>
            )}
          </div>
        </div>
      </div>

      {/* Auto-paused warning */}
      {drain.pausedReason && (
        <div className="flex items-start gap-3 p-4 border border-warning-6 bg-warning-2 rounded-lg">
          <TriangleWarning className="size-5 text-warning-11 shrink-0 mt-0.5" />
          <div className="flex flex-col gap-1 min-w-0">
            <h3 className="text-sm font-semibold text-warning-12">Drain auto-paused</h3>
            <p className="text-sm text-warning-11">
              The drain crossed the failure threshold and stopped forwarding to prevent further
              issues.
            </p>
            <pre className="text-xs text-warning-12 mt-1 font-mono bg-warning-3 px-2 py-1.5 rounded border border-warning-5 whitespace-pre-wrap break-all">
              {drain.pausedReason}
            </pre>
          </div>
        </div>
      )}

      {/* Stats grid */}
      <div className="grid grid-cols-2 md:grid-cols-4 gap-3">
        <StatCard
          label="Total delivered"
          value={drain.totalRecordsDelivered.toLocaleString()}
          icon={CircleCheck}
          tone="success"
        />
        <StatCard
          label="Last delivery"
          value={
            drain.lastDeliveryAt ? <TimestampInfo value={drain.lastDeliveryAt} /> : "Never"
          }
          tone="default"
        />
        <StatCard
          label="Recent failures"
          value={drain.consecutiveFailures.toString()}
          tone={drain.consecutiveFailures > 0 ? "warning" : "default"}
        />
        <StatCard
          label="Status"
          value={drain.enabled ? (drain.pausedReason ? "Paused" : "Active") : "Disabled"}
          tone={drain.enabled && !drain.pausedReason ? "success" : "default"}
        />
      </div>

      {/* Configuration */}
      <div className="flex flex-col gap-3">
        <h2 className="text-sm font-semibold text-gray-12">Configuration</h2>
        <div className="border border-grayA-4 rounded-[14px] overflow-hidden divide-y divide-grayA-4">
          <ConfigRow label="Sources" value={drain.sources.join(", ")} />
          <ConfigRow label="Environments" value={drain.environments.join(", ")} />
          <ConfigRow
            label="Apps"
            value={drain.apps.length === 0 ? "All apps in scope" : drain.apps.join(", ")}
          />
          <ConfigRow label="Provider" value={drain.provider} mono />
        </div>
      </div>

      {/* Last error if any */}
      {(drain.consecutiveFailures > 0 || drain.lastError) && (
        <div className="flex flex-col gap-3">
          <h2 className="text-sm font-semibold text-gray-12">Last error</h2>
          <div className="p-4 border border-error-6 bg-error-2 rounded-lg flex flex-col gap-2">
            <div className="flex items-center gap-2">
              <TriangleWarning className="size-4 text-error-11" />
              <span className="text-sm font-medium text-error-12">
                {drain.consecutiveFailures} consecutive{" "}
                {drain.consecutiveFailures === 1 ? "failure" : "failures"}
              </span>
            </div>
            <pre className="text-xs text-error-11 whitespace-pre-wrap font-mono bg-error-3 px-3 py-2 rounded border border-error-5 break-all">
              {drain.lastError ?? "(no error message)"}
            </pre>
          </div>
        </div>
      )}

      {/* Danger zone */}
      <div className="mt-6">
        <SettingsDangerZone>
          <SettingCard
            title="Delete log drain"
            description="The coordinator stops forwarding on the next tick. State and cursors are kept for audit but the drain stops appearing in the dashboard."
            border="both"
          >
            <div className="flex justify-end">
              <Button
                variant="outline"
                color="danger"
                onClick={() => {
                  if (window.confirm(`Delete log drain "${drain.name}"?`)) {
                    remove.mutate({ id: drain.id });
                  }
                }}
                loading={remove.isLoading}
              >
                <Trash className="size-4" />
                Delete drain
              </Button>
            </div>
          </SettingCard>
        </SettingsDangerZone>
      </div>
    </div>
  );
}

type LogDrain = {
  id: string;
  name: string;
  provider: "axiom";
  sources: string[];
  environments: string[];
  apps: string[];
  enabled: boolean;
  pausedReason: string | null;
  consecutiveFailures: number;
  lastError: string | null;
  lastDeliveryAt: number | null;
  totalRecordsDelivered: number;
};

function ProviderIcon({
  provider,
  className,
}: {
  provider: LogDrain["provider"];
  className?: string;
}) {
  switch (provider) {
    case "axiom":
      return <Database className={className} />;
  }
}

function Status({ drain }: { drain: LogDrain }) {
  if (drain.pausedReason) {
    return <Badge variant="warning">Auto-paused</Badge>;
  }
  if (!drain.enabled) {
    return <Badge variant="secondary">Paused</Badge>;
  }
  if (drain.consecutiveFailures > 0) {
    return (
      <Badge variant="warning">
        {drain.consecutiveFailures}{" "}
        {drain.consecutiveFailures === 1 ? "failure" : "failures"}
      </Badge>
    );
  }
  return <Badge variant="success">Healthy</Badge>;
}

function StatCard({
  label,
  value,
  icon: Icon,
  tone = "default",
}: {
  label: string;
  value: React.ReactNode;
  icon?: React.ComponentType<{ className?: string }>;
  tone?: "default" | "success" | "warning";
}) {
  const toneClasses = {
    default: "text-gray-12",
    success: "text-success-11",
    warning: "text-warning-11",
  };

  return (
    <div className="flex flex-col gap-2 p-4 border border-grayA-4 rounded-lg">
      <div className="flex items-center justify-between">
        <span className="text-xs font-medium text-gray-11 uppercase tracking-wide">{label}</span>
        {Icon && <Icon className={`size-4 ${toneClasses[tone]}`} />}
      </div>
      <div className={`text-lg font-semibold truncate ${toneClasses[tone]}`}>{value}</div>
    </div>
  );
}

function ConfigRow({
  label,
  value,
  mono,
}: {
  label: string;
  value: string;
  mono?: boolean;
}) {
  return (
    <div className="flex items-center justify-between gap-4 px-4 py-3">
      <span className="text-sm text-gray-11">{label}</span>
      <span className={`text-sm text-gray-12 ${mono ? "font-mono" : ""}`}>{value}</span>
    </div>
  );
}

function DetailSkeleton() {
  return (
    <div className="mx-auto max-w-4xl px-4 py-8 flex flex-col gap-6">
      <div className="h-4 w-32 bg-grayA-3 rounded animate-pulse" />
      <div className="flex items-start gap-3">
        <div className="size-10 bg-grayA-3 rounded-lg animate-pulse" />
        <div className="flex flex-col gap-2">
          <div className="h-7 w-48 bg-grayA-3 rounded animate-pulse" />
          <div className="h-4 w-32 bg-grayA-3 rounded animate-pulse" />
        </div>
      </div>
      <div className="grid grid-cols-2 md:grid-cols-4 gap-3">
        {[1, 2, 3, 4].map((i) => (
          <div key={i} className="h-20 border border-grayA-4 rounded-lg animate-pulse bg-grayA-2" />
        ))}
      </div>
    </div>
  );
}
