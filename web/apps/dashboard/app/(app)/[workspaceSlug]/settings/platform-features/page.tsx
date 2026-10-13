"use client";

import { canMutateWorkspace } from "@/lib/auth/roles";
import { platformPreviews } from "@/lib/flags/platform-previews";
import { useFlag, useFlags } from "@/lib/flags/provider";
import { trpc } from "@/lib/trpc/client";
import { getErrorMessage, getUnkeyClient } from "@/lib/unkey-client";
import { useWorkspace } from "@/providers/workspace-provider";
import { useMutation } from "@tanstack/react-query";
import {
  Badge,
  Button,
  PageContainer,
  PageHeader,
  PageHeaderContent,
  PageHeaderTitle,
  SettingCard,
  SettingCardGroup,
  SettingsGroup,
  SettingsGroupTitle,
  SettingsGroups,
  toast,
} from "@unkey/ui";
import { notFound } from "next/navigation";

export default function PlatformFeaturesPage() {
  const { isLoading, user } = useWorkspace();
  const canEnroll = canMutateWorkspace(user?.role);
  const enabled = useFlag("platformFeatures");
  const utils = trpc.useUtils();
  const features = useFlags();
  const previews = Object.entries(features.data ?? {})
    .flatMap(([slug, flag]) => {
      const preview = platformPreviews.find((preview) => preview.flag.slug === slug);
      return preview || user?.impersonator
        ? [{ ...flag, slug, title: preview?.title ?? slug }]
        : [];
    })
    .sort((a, b) => a.slug.localeCompare(b.slug));
  const update = useMutation({
    onMutate: () => features.cancel(),
    mutationFn: async (input: { slug: string; enabled: boolean }) => {
      await getUnkeyClient().workspace.updateFlag(input);
    },
    async onSuccess(_data, input) {
      await Promise.all([
        features.setCachedFlagValue(input),
        utils.workspace.getCurrent.invalidate(),
      ]);
      toast.success("Preview feature updated");
    },
    onError(error) {
      toast.error("We couldn't update the preview feature", {
        description: getErrorMessage(error),
      });
    },
  });

  if (!isLoading && !features.isLoading && !features.error && !enabled) {
    notFound();
  }

  return (
    <PageContainer>
      <PageHeader className="max-w-[920px]">
        <PageHeaderContent>
          <PageHeaderTitle>Platform preview</PageHeaderTitle>
        </PageHeaderContent>
      </PageHeader>
      <SettingsGroups>
        <SettingsGroup>
          <SettingsGroupTitle>Preview features</SettingsGroupTitle>
          <SettingCardGroup>
            {features.isLoading ? (
              <output className="block p-5 text-sm text-gray-11">Loading preview features…</output>
            ) : features.error ? (
              <div role="alert" className="flex items-center gap-4 p-5 text-sm text-gray-11">
                We couldn't load preview features.
                <Button variant="outline" onClick={() => features.refetch()}>
                  Retry
                </Button>
              </div>
            ) : previews.length === 0 ? (
              <p className="p-5 text-sm text-gray-11">
                No preview features are available for this workspace.
              </p>
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
                      <div id={`${feature.slug}-description`}>
                        {feature.description}
                        {blockedReason && <span className="block">{blockedReason}</span>}
                      </div>
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
