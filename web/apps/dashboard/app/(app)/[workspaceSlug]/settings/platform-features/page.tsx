"use client";

import { Switch } from "@/components/ui/switch";
import { canMutateWorkspace } from "@/lib/auth/roles";
import { useFlag } from "@/lib/flags/provider";
import { useWorkspace } from "@/providers/workspace-provider";
import { IconFlaskOutline18 } from "@unkey/icons";
import {
  Button,
  EmptyState,
  EmptyStateDescription,
  EmptyStateHeader,
  EmptyStateIcon,
  EmptyStateTitle,
  InfoTooltip,
  PageContainer,
  PageHeader,
  PageHeaderContent,
  PageHeaderTitle,
  SettingCard,
  SettingCardGroup,
  SettingsGroup,
  SettingsGroups,
} from "@unkey/ui";
import { notFound } from "next/navigation";
import { usePlatformPreviews } from "./use-platform-previews";

export default function PlatformFeaturesPage() {
  const { isLoading, user } = useWorkspace();
  const canEnroll = canMutateWorkspace(user?.role);
  const enabled = useFlag("platformFeatures");
  const { previews, update, isLoading: previewsLoading, error, refetch } = usePlatformPreviews();

  if (!isLoading && !previewsLoading && !error && !enabled) {
    notFound();
  }

  return (
    <PageContainer>
      <PageHeader className="max-w-[920px]">
        <PageHeaderContent>
          <PageHeaderTitle>Feature previews</PageHeaderTitle>
        </PageHeaderContent>
      </PageHeader>
      <SettingsGroups>
        <SettingsGroup>
          <SettingCardGroup>
            {previewsLoading ? (
              <output className="block p-5 text-sm text-gray-11">Loading preview features…</output>
            ) : error ? (
              <div role="alert" className="flex items-center gap-4 p-5 text-sm text-gray-11">
                We couldn't load preview features.
                <Button variant="outline" onClick={() => refetch()}>
                  Retry
                </Button>
              </div>
            ) : previews.length === 0 ? (
              <EmptyState frame="none">
                <EmptyStateIcon>
                  <IconFlaskOutline18 />
                </EmptyStateIcon>
                <EmptyStateHeader>
                  <EmptyStateTitle>No preview features available</EmptyStateTitle>
                  <EmptyStateDescription>
                    New features you can try early will show up here.
                  </EmptyStateDescription>
                </EmptyStateHeader>
              </EmptyState>
            ) : (
              previews.map((feature) => {
                const blockedReason =
                  feature.enabled && !feature.allowOptOut
                    ? "Unenrollment is not available for this feature."
                    : !feature.enabled && !feature.allowOptIn
                      ? "Enrollment is not available for this feature."
                      : undefined;
                const pendingEnabled =
                  update.isLoading && update.variables?.slug === feature.slug
                    ? update.variables.enabled
                    : undefined;
                return (
                  <SettingCard
                    key={feature.slug}
                    contentWidth="w-full @2xl:w-auto shrink-0 justify-end"
                    title={feature.title}
                    description={
                      <div id={`${feature.slug}-description`}>{feature.description}</div>
                    }
                  >
                    <InfoTooltip
                      content={<span id={`${feature.slug}-blocked-reason`}>{blockedReason}</span>}
                      disabled={!blockedReason}
                      asChild
                      position={{ side: "top" }}
                    >
                      <span
                        tabIndex={blockedReason ? 0 : undefined}
                        aria-describedby={
                          blockedReason ? `${feature.slug}-blocked-reason` : undefined
                        }
                      >
                        <Switch
                          checked={pendingEnabled ?? feature.enabled}
                          onCheckedChange={(enabled) =>
                            update.mutate({ slug: feature.slug, enabled })
                          }
                          aria-label={feature.title}
                          aria-describedby={`${feature.slug}-description`}
                          disabled={!canEnroll || Boolean(blockedReason) || update.isLoading}
                        />
                      </span>
                    </InfoTooltip>
                  </SettingCard>
                );
              })
            )}
          </SettingCardGroup>
        </SettingsGroup>
      </SettingsGroups>
    </PageContainer>
  );
}
