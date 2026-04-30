"use client";

import { trpc } from "@/lib/trpc/client";
import { BookBookmark, Database } from "@unkey/icons";
import { Badge, Button, Empty, TimestampInfo, toast } from "@unkey/ui";
import Link from "next/link";

type Props = {
  scope: "project" | "workspace";
  workspaceSlug: string;
  projectId?: string;
};

// LogDrainsList renders the list of drains with the same visual style as
// the deployments list — card-based rows with status badges, provider
// icons, and inline actions. Matches the pattern in deployments-card-list.tsx.
export function LogDrainsList({ scope, workspaceSlug, projectId }: Props) {
  const utils = trpc.useUtils();
  const list = trpc.deploy.logDrain.list.useQuery({ scope, projectId });
  const remove = trpc.deploy.logDrain.delete.useMutation({
    onSuccess: () => {
      toast.success("Log drain deleted");
      void utils.deploy.logDrain.list.invalidate();
    },
    onError: (err) => toast.error(err.message),
  });

  const baseHref =
    scope === "project"
      ? `/${workspaceSlug}/projects/${projectId}/log-drains`
      : `/${workspaceSlug}/settings/log-drains`;

  if (list.isLoading) {
    return <LogDrainsListSkeleton />;
  }

  if (list.isError) {
    return (
      <div className="border border-grayA-4 rounded-[14px] overflow-hidden">
        <div className="w-full flex justify-center items-center py-16 px-4">
          <Empty className="w-[400px] flex items-start">
            <Empty.Title>Failed to load log drains</Empty.Title>
            <Empty.Description className="text-left">{list.error.message}</Empty.Description>
          </Empty>
        </div>
      </div>
    );
  }

  const drains = list.data ?? [];

  return (
    <div className="flex flex-col gap-6">
      <div className="flex items-start justify-between">
        <div className="flex flex-col gap-1">
          <h1 className="font-semibold text-gray-12 text-lg">Log Drains</h1>
          <p className="text-sm text-gray-11 max-w-2xl">
            Forward runtime and request logs from your deployments to Axiom. Drains are scoped{" "}
            {scope === "project" ? "to this project" : "across every project in the workspace"}.
          </p>
        </div>
        <Link href={`${baseHref}/new`}>
          <Button variant="primary" size="md">
            New drain
          </Button>
        </Link>
      </div>

      {drains.length === 0 ? (
        <div className="border border-grayA-4 rounded-[14px] overflow-hidden">
          <div className="w-full flex justify-center items-center py-16 px-4">
            <Empty className="w-[400px] flex items-start">
              <Empty.Icon className="w-auto" />
              <Empty.Title>No Log Drains Found</Empty.Title>
              <Empty.Description className="text-left">
                Connect a provider once and Unkey will forward every deployment&apos;s logs to it.
              </Empty.Description>
              <Empty.Actions className="mt-4 justify-start">
                <Link href={`${baseHref}/new`}>
                  <Button size="md" variant="primary">
                    Create your first drain
                  </Button>
                </Link>
                <a
                  href="https://www.unkey.com/docs/observability/log-drains"
                  target="_blank"
                  rel="noopener noreferrer"
                >
                  <Button size="md" variant="outline">
                    <BookBookmark />
                    Learn about Log Drains
                  </Button>
                </a>
              </Empty.Actions>
            </Empty>
          </div>
        </div>
      ) : (
        <div className="border border-grayA-4 rounded-[14px] overflow-hidden divide-y divide-grayA-4">
          {drains.map((drain) => (
            <LogDrainRow
              key={drain.id}
              drain={drain}
              href={`${baseHref}/${drain.id}`}
              onDelete={() => {
                if (
                  window.confirm(
                    "Delete this log drain? The coordinator stops forwarding on the next tick.",
                  )
                ) {
                  remove.mutate({ id: drain.id });
                }
              }}
            />
          ))}
        </div>
      )}
    </div>
  );
}

type LogDrain = {
  id: string;
  name: string;
  provider: "axiom";
  sources: string[];
  environments: string[];
  enabled: boolean;
  pausedReason: string | null;
  consecutiveFailures: number;
  lastError: string | null;
  lastDeliveryAt: number | null;
  totalRecordsDelivered: number;
};

function LogDrainRow({
  drain,
  href,
  onDelete,
}: {
  drain: LogDrain;
  href: string;
  onDelete: () => void;
}) {
  return (
    <div className="relative flex flex-col md:flex-row md:items-center px-4 py-3 gap-3 md:gap-0 transition-colors hover:bg-grayA-2">
      <Link href={href} className="absolute inset-0 z-10" aria-label={`Log drain ${drain.name}`} />

      {/* Identity + Status */}
      <div className="flex items-center justify-between md:contents">
        <div className="md:w-[28%] md:shrink-0 flex flex-col gap-1 min-w-0">
          <div className="flex items-center gap-2">
            <ProviderIcon provider={drain.provider} />
            <span className="font-medium text-[13px] text-accent-12 truncate">{drain.name}</span>
          </div>
          <div className="flex items-center gap-1.5 text-[12px] text-gray-11">
            <span className="font-mono">{drain.provider}</span>
          </div>
        </div>

        {/* Status */}
        <div className="md:w-[15%] md:shrink-0 flex md:justify-start">
          <StatusBadge drain={drain} />
        </div>
      </div>

      {/* Sources */}
      <div className="md:w-[20%] md:shrink-0 flex flex-col gap-0.5 text-[12px]">
        <span className="text-gray-12 font-medium">Sources</span>
        <span className="text-gray-11">{drain.sources.join(", ")}</span>
      </div>

      {/* Environments */}
      <div className="md:w-[15%] md:shrink-0 flex flex-col gap-0.5 text-[12px]">
        <span className="text-gray-12 font-medium">Environments</span>
        <span className="text-gray-11">{drain.environments.join(", ")}</span>
      </div>

      {/* Last delivery */}
      <div className="md:w-[17%] md:shrink-0 flex flex-col gap-0.5 text-[12px]">
        <span className="text-gray-12 font-medium">Last delivery</span>
        <span className="text-gray-11">
          {drain.lastDeliveryAt ? (
            <TimestampInfo value={drain.lastDeliveryAt} />
          ) : (
            <span className="italic">Never</span>
          )}
        </span>
      </div>

      {/* Actions */}
      <div className="md:w-[5%] md:shrink-0 flex md:justify-end relative z-20">
        <Button
          variant="outline"
          size="sm"
          onClick={(e) => {
            e.preventDefault();
            e.stopPropagation();
            onDelete();
          }}
        >
          Delete
        </Button>
      </div>
    </div>
  );
}

function ProviderIcon({ provider }: { provider: LogDrain["provider"] }) {
  const iconClass = "size-4 text-gray-11";
  switch (provider) {
    case "axiom":
      return <Database className={iconClass} />;
  }
}

function StatusBadge({ drain }: { drain: LogDrain }) {
  if (drain.pausedReason) {
    return <Badge variant="warning">Auto-paused</Badge>;
  }
  if (!drain.enabled) {
    return <Badge variant="secondary">Paused</Badge>;
  }
  if (drain.consecutiveFailures > 0) {
    return (
      <Badge variant="warning">
        {drain.consecutiveFailures} {drain.consecutiveFailures === 1 ? "failure" : "failures"}
      </Badge>
    );
  }
  return <Badge variant="success">Healthy</Badge>;
}

function LogDrainsListSkeleton() {
  return (
    <div className="flex flex-col gap-6">
      <div className="flex items-start justify-between">
        <div className="flex flex-col gap-2">
          <div className="h-5 w-32 bg-grayA-3 rounded animate-pulse" />
          <div className="h-4 w-96 bg-grayA-3 rounded animate-pulse" />
        </div>
        <div className="h-9 w-24 bg-grayA-3 rounded animate-pulse" />
      </div>
      <div className="border border-grayA-4 rounded-[14px] overflow-hidden divide-y divide-grayA-4">
        {[1, 2, 3].map((i) => (
          <div key={i} className="px-4 py-3 flex items-center gap-4">
            <div className="w-[28%] flex flex-col gap-1.5">
              <div className="h-4 w-32 bg-grayA-3 rounded animate-pulse" />
              <div className="h-3 w-16 bg-grayA-3 rounded animate-pulse" />
            </div>
            <div className="w-[15%] h-5 bg-grayA-3 rounded animate-pulse" />
            <div className="w-[20%] h-4 bg-grayA-3 rounded animate-pulse" />
            <div className="w-[15%] h-4 bg-grayA-3 rounded animate-pulse" />
            <div className="w-[17%] h-4 bg-grayA-3 rounded animate-pulse" />
            <div className="w-[5%] h-7 bg-grayA-3 rounded animate-pulse" />
          </div>
        ))}
      </div>
    </div>
  );
}
