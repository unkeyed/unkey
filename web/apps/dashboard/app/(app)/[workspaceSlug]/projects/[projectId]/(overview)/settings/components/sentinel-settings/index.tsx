"use client";

import { trpc } from "@/lib/trpc/client";
import { mapRegionToFlag } from "@/lib/trpc/routers/deploy/network/utils";
import { formatCpuParts, formatMemoryParts } from "@/lib/utils/deployment-formatters";
import { Microchip, Minus, Plus, Refresh3 } from "@unkey/icons";
import {
  Button,
  Loading,
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
  toast,
} from "@unkey/ui";
import { useEffect, useMemo, useState } from "react";
import { RegionFlag } from "../../../../components/region-flag";
import { useProjectData } from "../../../data-provider";

type SentinelRow = {
  sentinelId: string;
  environmentId: string;
  environmentSlug: string;
  regionId: string;
  regionName: string;
  desiredReplicas: number;
  availableReplicas: number;
  health: "unknown" | "paused" | "healthy" | "unhealthy";
  desiredState: "running" | "standby" | "archived";
  deployStatus: "idle" | "progressing" | "ready" | "failed";
  tierId: string;
  tierVersion: string;
  cpuMillicores: number;
  memoryMib: number;
  minReplicas: number;
};

const HEALTH_DOT: Record<SentinelRow["health"], string> = {
  healthy: "bg-success-9",
  paused: "bg-warning-9",
  unhealthy: "bg-error-9",
  unknown: "bg-gray-7",
};

const HEALTH_LABEL: Record<SentinelRow["health"], string> = {
  healthy: "Healthy",
  paused: "Paused",
  unhealthy: "Unhealthy",
  unknown: "Unknown",
};

const ENV_SUBTITLE: Record<string, string> = {
  production: "User-facing traffic. Changes roll out live.",
  preview: "Sentinels for branch deploys and pre-production testing.",
};

const formatCpu = (millicores: number) => {
  const { value, unit } = formatCpuParts(millicores);
  return unit ? `${value} ${unit}` : value;
};

const formatMemory = (mib: number) => {
  const { value, unit } = formatMemoryParts(mib);
  return unit ? `${value} ${unit}` : value;
};

// While any sentinel is progressing we poll the list so the UI unlocks as
// soon as krane reports the rollout converged. 3s balances responsiveness
// against server load; rollouts typically take 15-60s.
const PROGRESSING_POLL_MS = 3_000;
const MAX_ADDITIONAL_REPLICAS = 24;

export const SentinelSettings = () => {
  const { projectId } = useProjectData();
  const [anyProgressing, setAnyProgressing] = useState(false);
  const sentinelsQuery = trpc.deploy.sentinel.list.useQuery(
    { projectId },
    { refetchInterval: anyProgressing ? PROGRESSING_POLL_MS : false },
  );
  const tiersQuery = trpc.deploy.sentinel.listTiers.useQuery();

  const sentinels = sentinelsQuery.data ?? [];
  const tiers = tiersQuery.data ?? [];

  useEffect(() => {
    setAnyProgressing(sentinels.some((s) => s.deployStatus === "progressing"));
  }, [sentinels]);

  // Group sentinels by environment. Production first (it's the load-bearing
  // one), preview second (next-most-common), then everything else alphabetical.
  const grouped = useMemo(() => {
    const byEnv = new Map<string, SentinelRow[]>();
    for (const s of sentinels) {
      const bucket = byEnv.get(s.environmentSlug) ?? [];
      bucket.push(s);
      byEnv.set(s.environmentSlug, bucket);
    }
    for (const [, bucket] of byEnv) {
      bucket.sort((a, b) => a.regionName.localeCompare(b.regionName));
    }
    const rank = (slug: string) => (slug === "production" ? 0 : slug === "preview" ? 1 : 2);
    return [...byEnv.entries()].sort(([a], [b]) => {
      const diff = rank(a) - rank(b);
      return diff !== 0 ? diff : a.localeCompare(b);
    });
  }, [sentinels]);

  if (sentinelsQuery.isLoading || tiersQuery.isLoading) {
    return (
      <div className="flex items-center justify-center p-6">
        <Loading />
      </div>
    );
  }

  if (sentinels.length === 0) {
    return (
      <div className="border border-grayA-4 rounded-[14px] px-6 py-10 flex flex-col items-center gap-1 text-center">
        <span className="text-[13px] text-gray-12 font-medium">No sentinels yet</span>
        <span className="text-[12px] text-gray-11">
          Sentinels are provisioned automatically on your first deploy.
        </span>
      </div>
    );
  }

  return (
    <div className="flex flex-col gap-4">
      {grouped.map(([envSlug, rows]) => {
        const subtitle =
          ENV_SUBTITLE[envSlug] ?? "Custom environment. Sentinels run per configured region.";
        return (
          <div
            key={envSlug}
            className="border border-grayA-4 rounded-[14px] overflow-hidden dark:bg-black bg-white"
          >
            <div className="flex items-start justify-between px-4 py-3 bg-grayA-2 border-b border-grayA-4 gap-4">
              <div className="flex items-start gap-3 min-w-0">
                <div className="mt-0.5 size-7 rounded-[8px] bg-grayA-3 border border-grayA-4 flex items-center justify-center shrink-0">
                  <Microchip className="text-gray-11 size-3.5" />
                </div>
                <div className="flex flex-col gap-0.5 min-w-0">
                  <span className="text-[13px] font-medium text-gray-12 capitalize leading-none">
                    {envSlug}
                  </span>
                  <span className="text-[12px] text-gray-10 leading-tight truncate">
                    {subtitle}
                  </span>
                </div>
              </div>
              <span className="shrink-0 inline-flex items-center h-6 px-2 rounded-md bg-grayA-3 border border-grayA-4 text-[11px] text-gray-11 font-mono tabular-nums">
                {rows.length} {rows.length === 1 ? "region" : "regions"}
              </span>
            </div>
            <div className="divide-y divide-grayA-3">
              {rows.map((row) => (
                <SentinelRowEditor
                  key={row.sentinelId}
                  row={row}
                  tiers={tiers}
                  onSaved={() => sentinelsQuery.refetch()}
                />
              ))}
            </div>
          </div>
        );
      })}
    </div>
  );
};

type SentinelRowEditorProps = {
  row: SentinelRow;
  tiers: Array<{
    tierId: string;
    version: string;
    cpuMillicores: number;
    memoryMib: number;
    pricePerSecond: string;
  }>;
  onSaved: () => void;
};

const SentinelRowEditor = ({ row, tiers, onSaved }: SentinelRowEditorProps) => {
  const isRolling = row.deployStatus === "progressing";
  const [selectedKey, setSelectedKey] = useState(`${row.tierId}::${row.tierVersion}`);
  // Users edit the *additional* replica count they want on top of the
  // environment's baseline (prod=3, others=1). minReplicas is fixed by
  // policy and not user-editable; the number sent to the backend is
  // minReplicas + additional.
  const [additional, setAdditional] = useState(Math.max(0, row.desiredReplicas - row.minReplicas));

  const currentKey = `${row.tierId}::${row.tierVersion}`;
  useEffect(() => {
    setSelectedKey(currentKey);
  }, [currentKey]);
  useEffect(() => {
    setAdditional(Math.max(0, row.desiredReplicas - row.minReplicas));
  }, [row.desiredReplicas, row.minReplicas]);

  const onMutate = {
    onSuccess: () => {
      toast.success(`Rolling out ${row.environmentSlug} / ${row.regionName}...`);
      onSaved();
    },
    onError: (err: { message: string }) => {
      toast.error(err.message);
    },
  };
  const changeTier = trpc.deploy.sentinel.changeTier.useMutation(onMutate);
  const changeReplicas = trpc.deploy.sentinel.changeReplicas.useMutation(onMutate);

  const tierOptions = useMemo(
    () =>
      tiers
        .map((t) => ({ ...t, key: `${t.tierId}::${t.version}` }))
        .sort((a, b) => a.cpuMillicores - b.cpuMillicores),
    [tiers],
  );

  const selectedTier = tierOptions.find((t) => t.key === selectedKey);
  const tierDirty = selectedKey !== currentKey;
  const desiredReplicas = row.minReplicas + additional;
  const replicasDirty = desiredReplicas !== row.desiredReplicas;
  const isDirty = tierDirty || replicasDirty;
  const isSaving = changeTier.isLoading || changeReplicas.isLoading;
  const locked = isRolling || isSaving;

  const onSave = () => {
    if (tierDirty) {
      const [tierId, tierVersion] = selectedKey.split("::");
      changeTier.mutate({ sentinelId: row.sentinelId, tierId, tierVersion });
    }
    if (replicasDirty) {
      changeReplicas.mutate({ sentinelId: row.sentinelId, desiredReplicas });
    }
  };

  return (
    <div
      className={`relative flex items-center gap-5 px-4 h-16 ${
        isDirty
          ? "before:absolute before:inset-y-2 before:left-0 before:w-0.5 before:rounded-r-full before:bg-infoA-9"
          : ""
      }`}
    >
      {/* Identity. Flag sits in a subtle rounded container so the row has
          a clear anchor on the left; region + inline status pill share one
          line, with the availability ratio as the muted subline. */}
      <div className="flex items-center gap-3 w-[220px] shrink-0">
        <div className="size-7 rounded-[8px] bg-grayA-3 border border-grayA-4 flex items-center justify-center shrink-0">
          <RegionFlag
            flagCode={mapRegionToFlag(row.regionName)}
            size="xs"
            className="[&_img]:size-4"
          />
        </div>
        <div className="flex flex-col gap-1 min-w-0">
          <div className="flex items-center gap-2 leading-none">
            <span className="text-[13px] font-medium text-gray-12 font-mono truncate">
              {row.regionName}
            </span>
            {isRolling ? (
              <span className="inline-flex items-center gap-1 h-4 px-1.5 rounded-[4px] bg-grayA-3 border border-grayA-4 text-[10px] text-gray-11">
                <Refresh3 className="size-2.5! animate-spin" iconSize="sm-regular" />
                Rolling out
              </span>
            ) : (
              <span className="inline-flex items-center gap-1 h-4 px-1.5 rounded-[4px] bg-grayA-3 border border-grayA-4 text-[10px] text-gray-11">
                <span className={`inline-block size-1.5 rounded-full ${HEALTH_DOT[row.health]}`} />
                {HEALTH_LABEL[row.health]}
              </span>
            )}
          </div>
          <span className="text-[11px] text-gray-10 font-mono tabular-nums leading-none">
            {row.availableReplicas}/{row.desiredReplicas} available
          </span>
        </div>
      </div>

      {/* Tier select. Renders tier id in the trigger with CPU/memory envelope
          as a muted mono pair on the right. */}
      <div className="flex-1 min-w-0">
        <Select value={selectedKey} onValueChange={setSelectedKey} disabled={locked}>
          <SelectTrigger wrapperClassName="w-full max-w-[260px]">
            <SelectValue>
              <div className="flex items-center justify-between gap-3 w-full pr-1">
                <span className="text-[13px] font-medium text-gray-12">
                  {selectedTier?.tierId ?? row.tierId}
                </span>
                <span className="text-[11px] text-gray-10 font-mono tabular-nums">
                  {formatCpu(selectedTier?.cpuMillicores ?? row.cpuMillicores)}
                  {" / "}
                  {formatMemory(selectedTier?.memoryMib ?? row.memoryMib)}
                </span>
              </div>
            </SelectValue>
          </SelectTrigger>
          <SelectContent>
            {tierOptions.map((t) => (
              <SelectItem key={t.key} value={t.key}>
                <div className="flex items-center justify-between gap-6 w-full">
                  <span className="text-[13px] font-medium text-gray-12">{t.tierId}</span>
                  <span className="text-[11px] text-gray-10 font-mono tabular-nums">
                    {formatCpu(t.cpuMillicores)} / {formatMemory(t.memoryMib)}
                  </span>
                </div>
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>

      {/* Replicas stepper. No label, no readout. The "+N" format leans on
          the environment floor being implicit (prod=3, others=1) while
          still making zero feel like zero. Icons instead of glyphs for
          pixel-perfect centering. */}
      <div
        className="flex items-center h-8 rounded-lg border border-grayA-5 bg-white dark:bg-black data-[disabled=true]:opacity-50 shrink-0"
        data-disabled={locked}
      >
        <button
          type="button"
          aria-label="Fewer replicas"
          disabled={locked || additional === 0}
          onClick={() => setAdditional((n) => Math.max(0, n - 1))}
          className="size-8 flex items-center justify-center text-gray-11 hover:text-gray-12 hover:bg-grayA-3 disabled:text-gray-7 disabled:hover:bg-transparent disabled:cursor-not-allowed transition-colors rounded-l-lg"
        >
          <Minus className="size-3" iconSize="sm-regular" />
        </button>
        <div className="w-10 h-full flex items-center justify-center border-x border-grayA-4">
          <input
            type="text"
            inputMode="numeric"
            pattern="[0-9]*"
            aria-label="Additional replicas"
            value={`+${additional}`}
            disabled={locked}
            onChange={(e) => {
              const digits = e.target.value.replace(/\D/g, "");
              if (digits === "") {
                setAdditional(0);
                return;
              }
              const next = Number.parseInt(digits, 10);
              if (!Number.isNaN(next)) {
                setAdditional(Math.min(MAX_ADDITIONAL_REPLICAS, Math.max(0, next)));
              }
            }}
            className="w-full text-center text-[13px] font-mono tabular-nums bg-transparent text-gray-12 focus:outline-none"
          />
        </div>
        <button
          type="button"
          aria-label="More replicas"
          disabled={locked || additional >= MAX_ADDITIONAL_REPLICAS}
          onClick={() => setAdditional((n) => Math.min(MAX_ADDITIONAL_REPLICAS, n + 1))}
          className="size-8 flex items-center justify-center text-gray-11 hover:text-gray-12 hover:bg-grayA-3 disabled:text-gray-7 disabled:hover:bg-transparent disabled:cursor-not-allowed transition-colors rounded-r-lg"
        >
          <Plus className="size-3" iconSize="sm-regular" />
        </button>
      </div>

      <Button
        size="sm"
        variant={isDirty ? "primary" : "outline"}
        disabled={!isDirty || locked}
        loading={isSaving}
        onClick={onSave}
        className="shrink-0"
      >
        Save
      </Button>
    </div>
  );
};
