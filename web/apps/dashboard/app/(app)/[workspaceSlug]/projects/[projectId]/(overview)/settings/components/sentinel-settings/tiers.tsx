"use client";

import { SENTINEL_TIERS, SENTINEL_TIERS_BY_ID } from "@/lib/constants/sentinel-tiers";
import { trpc } from "@/lib/trpc/client";
import { Bolt, Check } from "@unkey/icons";
import { SettingCard } from "@unkey/ui";
import { cn } from "@unkey/ui/src/lib/utils";
import { useEnvironmentSettings } from "../../environment-provider";

function formatCpu(millicores: number) {
  if (millicores >= 1000) {
    return `${millicores / 1000} vCPU`;
  }
  return `${millicores}m`;
}

function formatMemory(mib: number) {
  if (mib >= 1024) {
    return `${(mib / 1024).toFixed(0)} GB`;
  }
  return `${mib} MB`;
}

export function SentinelTiers() {
  const { settings } = useEnvironmentSettings();

  const { data: sentinel } = trpc.deploy.environmentSettings.sentinel.getByEnvironment.useQuery({
    environmentId: settings.environmentId,
  });

  const utils = trpc.useUtils();
  const updateTier = trpc.deploy.environmentSettings.sentinel.updateTier.useMutation({
    onSuccess: () => {
      utils.deploy.environmentSettings.sentinel.getByEnvironment.invalidate();
    },
  });

  if (!sentinel) {
    return null;
  }

  const activeTier = SENTINEL_TIERS_BY_ID[sentinel.sentinelTierId];

  const displayValue = activeTier ? (
    <span className="flex items-center gap-2 text-gray-11 text-xs">
      <span className="font-medium text-gray-12">{activeTier.name}</span>
      <span>{formatCpu(activeTier.cpuMillicores)}</span>
      <span>{formatMemory(activeTier.memoryMib)}</span>
    </span>
  ) : (
    "Standard"
  );

  return (
    <SettingCard
      className="px-4 py-[18px]"
      icon={<Bolt className="text-gray-12" iconSize="xl-medium" />}
      title="Sentinel size"
      description={
        <>
          Resource tier for your sentinel proxies.{" "}
          <a
            href="https://unkey.com/pricing"
            target="_blank"
            rel="noopener noreferrer"
            className="text-accent-11 hover:underline"
          >
            See pricing
          </a>
        </>
      }
      contentWidth="w-full lg:w-[320px] justify-end"
      expandable={
        <div className="bg-grayA-2 rounded-b-xl p-4">
          <table className="w-full text-xs">
            <thead>
              <tr className="text-gray-9 border-b border-grayA-3">
                <th className="text-left font-normal px-3 py-2">Tier</th>
                <th className="text-left font-normal px-3 py-2">vCPU</th>
                <th className="text-left font-normal px-3 py-2">Memory</th>
                <th className="w-8" />
              </tr>
            </thead>
            <tbody>
              {SENTINEL_TIERS.map((tier) => {
                const isActive = tier.id === sentinel.sentinelTierId;
                return (
                  <tr
                    key={tier.id}
                    onClick={() => {
                      if (!isActive) {
                        updateTier.mutate({ sentinelId: sentinel.id, tierId: tier.id });
                      }
                    }}
                    className={cn(
                      "cursor-pointer transition-colors",
                      isActive ? "bg-accent-3 text-accent-11" : "hover:bg-grayA-3 text-gray-11",
                    )}
                  >
                    <td
                      className={cn(
                        "px-3 py-2.5 font-medium",
                        isActive ? "text-accent-12" : "text-gray-12",
                      )}
                    >
                      {tier.name}
                    </td>
                    <td className={cn("px-3 py-2.5", isActive && "font-medium text-accent-12")}>
                      {formatCpu(tier.cpuMillicores)}
                    </td>
                    <td className={cn("px-3 py-2.5", isActive && "font-medium text-accent-12")}>
                      {formatMemory(tier.memoryMib)}
                    </td>
                    <td className="px-3 py-2.5 w-8">
                      {isActive && <Check iconSize="sm-regular" className="text-accent-11" />}
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>
      }
    >
      {displayValue}
    </SettingCard>
  );
}
