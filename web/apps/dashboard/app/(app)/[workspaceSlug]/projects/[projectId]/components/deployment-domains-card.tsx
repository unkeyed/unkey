"use client";

import type { Domain } from "@/lib/collections";
import { cn } from "@/lib/utils";
import { ChevronDown, Cube, Earth, Link4 } from "@unkey/icons";
import {
  Button,
  CopyButton,
  Popover,
  PopoverContent,
  PopoverTrigger,
  SettingCard,
  SettingCardGroup,
} from "@unkey/ui";
import { type ReactNode, useState } from "react";
import { useProjectData } from "../(overview)/data-provider";
import { useDeployment } from "../(overview)/deployments/[deploymentId]/layout-provider";
import { SettingsGroup } from "../(overview)/settings/components/shared/settings-group";

export function DeploymentDomainsCard({
  emptyState,
  glow,
  domainFilter,
}: { emptyState?: ReactNode; glow?: boolean; domainFilter?: (d: Domain) => boolean }) {
  const [urlsOpen, setUrlsOpen] = useState(false);
  const { deployment } = useDeployment();
  const { getDomainsForDeployment, isDomainsLoading, project } = useProjectData();

  const domainsForDeployment = getDomainsForDeployment(deployment.id);
  const filtered = domainFilter ? domainsForDeployment.filter(domainFilter) : domainsForDeployment;
  const sortedDomains = [...filtered].sort((a, b) =>
    a.fullyQualifiedDomainName.localeCompare(b.fullyQualifiedDomainName),
  );
  const environmentDomain = sortedDomains.find((d) => d.sticky === "environment");
  const primaryDomain = environmentDomain ?? sortedDomains[0];
  const additionalDomains = primaryDomain
    ? sortedDomains.filter((d) => d.id !== primaryDomain.id)
    : [];

  if (!isDomainsLoading && sortedDomains.length === 0) {
    return emptyState ?? null;
  }

  return (
    <SettingsGroup
      icon={<Earth iconSize="md-medium" />}
      title={<span className="font-medium text-gray-12 text-[13px] leading-4">Domains</span>}
      hideChevron
    >
      <SettingCardGroup>
        {isDomainsLoading ? (
          <SettingCard
            icon={
              <div className="w-full h-full rounded-[10px] flex items-center justify-center shrink-0 dark:ring-1 dark:ring-gray-4 dark:shadow-none shadow-sm shadow-grayA-8/20">
                <Earth iconSize="sm-medium" className="size-[18px]" />
              </div>
            }
            title={<div className="h-4 w-36 bg-grayA-3 rounded animate-pulse" />}
            description="Loading domains..."
          />
        ) : (
          <SettingCard
            icon={
              glow ? (
                <div className="relative w-full h-full">
                  <div
                    className={cn(
                      "absolute inset-[-4px] rounded-[10px] blur-[14px]",
                      "bg-gradient-to-l from-feature-8 to-info-9",
                      "animate-pulse opacity-20",
                    )}
                  />
                  <div
                    className={cn(
                      "w-full h-full rounded-[10px] flex items-center justify-center shrink-0 dark:ring-1 dark:ring-gray-4 dark:shadow-none relative dark:bg-white dark:text-black bg-black text-white",
                    )}
                  >
                    <Cube iconSize="md-medium" className="size-[18px]" />
                  </div>
                </div>
              ) : (
                <div className="w-full h-full rounded-[10px] flex items-center justify-center shrink-0 dark:ring-1 dark:ring-gray-4 dark:shadow-none shadow-sm shadow-grayA-8/20">
                  <Cube iconSize="md-medium" className="size-[18px]" />
                </div>
              )
            }
            title={project?.name}
            description={
              <div className="flex items-center justify-center gap-2 ">
                <a
                  href={`https://${primaryDomain.fullyQualifiedDomainName}`}
                  target="_blank"
                  rel="noopener noreferrer"
                  className="text-gray-10 text-xs hover:underline decoration-dashed underline-offset-2 transition-all"
                >
                  {primaryDomain.fullyQualifiedDomainName}
                </a>
                {additionalDomains.length > 0 && (
                  <button
                    type="button"
                    className="rounded-full px-1.5 py-0.5 bg-grayA-3 text-gray-12 text-xs leading-[18px] font-mono tabular-nums hover:bg-grayA-4 transition-colors cursor-pointer"
                    onClick={() => setUrlsOpen(true)}
                  >
                    +{additionalDomains.length}
                  </button>
                )}
              </div>
            }
            contentWidth="w-fit"
          >
            <div className="flex items-center gap-2">
              {additionalDomains.length > 0 && (
                <Popover open={urlsOpen} onOpenChange={setUrlsOpen}>
                  <PopoverTrigger asChild>
                    <Button
                      className="text-gray-12 font-medium bg-grayA-2 rounded-[8px]"
                      variant="outline"
                    >
                      Show URLs
                      <ChevronDown className="text-gray-9 !size-3" iconSize="sm-regular" />
                    </Button>
                  </PopoverTrigger>
                  <PopoverContent side="bottom" align="end" className="p-0 max-w-[300px]">
                    {sortedDomains.map((d) => (
                      <div
                        key={d.id}
                        className="flex items-center justify-left w-full h-10 border-b border-gray-4 px-3 py-[14px] gap-2"
                      >
                        <Link4 className="text-gray-9 !size-3 shrink-0" iconSize="sm-regular" />
                        <a
                          href={`https://${d.fullyQualifiedDomainName}`}
                          target="_blank"
                          rel="noopener noreferrer"
                          className="transition-all hover:underline decoration-dashed underline-offset-2 max-w-[250px] truncate font-medium text-xs"
                        >
                          {d.fullyQualifiedDomainName}
                        </a>
                        <CopyButton
                          value={`https://${d.fullyQualifiedDomainName}`}
                          variant="ghost"
                          toastMessage={d.fullyQualifiedDomainName}
                        />
                      </div>
                    ))}
                  </PopoverContent>
                </Popover>
              )}
            </div>
          </SettingCard>
        )}
      </SettingCardGroup>
    </SettingsGroup>
  );
}
