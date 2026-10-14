"use client";

import { canMutateWorkspace } from "@/lib/auth/roles";
import { useFlag } from "@/lib/flags/provider";
import { useWorkspace } from "@/providers/workspace-provider";
import { IconSparkle3Outline18 } from "@unkey/icons";
import {
  Badge,
  Button,
  EmptyState,
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
          <PageHeaderTitle>Platform previews</PageHeaderTitle>
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
                  <IconSparkle3Outline18 />
                </EmptyStateIcon>
                <EmptyStateHeader>
                  <EmptyStateTitle>No preview features available</EmptyStateTitle>
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
                return (
                  <SettingCard
                    key={feature.slug}
                    contentWidth="w-full @2xl:w-auto shrink-0 justify-end"
                    title={
                      <div className="flex flex-wrap items-center gap-2">
                        <span className="break-words">{feature.title}</span>
                        <Badge variant={feature.enabled ? "success" : "primary"}>
                          {feature.enabled ? "Enrolled" : "Not enrolled"}
                        </Badge>
                      </div>
                    }
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
                        <Button
                          variant="outline"
                          color={feature.enabled ? "danger" : "default"}
                          size="sm"
                          aria-label={`${feature.enabled ? "Unenroll from" : "Enroll in"} ${feature.title}`}
                          aria-describedby={`${feature.slug}-description`}
                          disabled={!canEnroll || Boolean(blockedReason) || update.isLoading}
                          loading={update.isLoading && update.variables?.slug === feature.slug}
                          loadingLabel={feature.enabled ? "Unenrolling" : "Enrolling"}
                          onClick={() =>
                            update.mutate({ slug: feature.slug, enabled: !feature.enabled })
                          }
                        >
                          {feature.enabled ? "Unenroll" : "Enroll"}
                        </Button>
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
