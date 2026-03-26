"use client";

import { Combobox } from "@/components/ui/combobox";
import { trpc } from "@/lib/trpc/client";
import { CodeBranch } from "@unkey/icons";
import { Button, InfoTooltip, SettingCard, toast } from "@unkey/ui";
import { useMemo, useState } from "react";
import { useProjectData } from "../../../data-provider";
import { SelectedConfig } from "../shared/selected-config";

export const DefaultBranch = () => {
  const { projectId } = useProjectData();
  const utils = trpc.useUtils();

  const { data, isLoading } = trpc.github.getInstallations.useQuery(
    { projectId },
    { staleTime: 0, refetchOnWindowFocus: true },
  );

  const appId = data?.appId;
  const repoConnection = data?.repoConnection;
  const currentDefaultBranch = data?.defaultBranch ?? "main";

  const [selectedBranch, setSelectedBranch] = useState<string | null>(null);

  const repoFullName = repoConnection?.repositoryFullName;
  const installationId = repoConnection?.installationId;
  const [owner, repo] = repoFullName?.split("/") ?? [];

  const { data: details } = trpc.github.getRepositoryDetails.useQuery(
    {
      projectId,
      installationId: installationId ?? 0,
      owner: owner ?? "",
      repo: repo ?? "",
      defaultBranch: currentDefaultBranch,
    },
    {
      enabled: Boolean(installationId && owner && repo),
      refetchOnWindowFocus: false,
    },
  );

  const branchOptions = useMemo(() => {
    const branches = details?.branches ?? [];
    // Sort: branches with a last push date come first (most recent first),
    // then alphabetically for branches without activity data.
    const sorted = [...branches].sort((a, b) => {
      if (a.lastPushDate && b.lastPushDate) {
        return new Date(b.lastPushDate).getTime() - new Date(a.lastPushDate).getTime();
      }
      if (a.lastPushDate) {
        return -1;
      }
      if (b.lastPushDate) {
        return 1;
      }
      return a.name.localeCompare(b.name);
    });
    return sorted.map((b) => ({ label: b.name, value: b.name }));
  }, [details?.branches]);

  const updateMutation = trpc.github.updateDefaultBranch.useMutation({
    onSuccess: async () => {
      toast.success("Default branch updated");
      setSelectedBranch(null);
      await utils.github.getInstallations.invalidate();
    },
    onError: (error) => {
      toast.error(error.message);
    },
  });

  if (isLoading) {
    return (
      <SettingCard
        className="px-4 py-[18px]"
        icon={<CodeBranch className="text-gray-12" iconSize="xl-regular" />}
        title="Production branch"
        description="Branch that triggers production deployments"
        contentWidth="w-full lg:w-[320px] justify-end"
        chevronState="disabled"
      >
        <div className="h-7 w-24 bg-grayA-3 rounded animate-pulse" />
      </SettingCard>
    );
  }

  // Only show when a repo is connected
  if (!repoConnection || !appId) {
    return null;
  }

  const effectiveBranch = selectedBranch ?? currentDefaultBranch;
  const hasChanges = selectedBranch !== null && selectedBranch !== currentDefaultBranch;

  const handleSave = () => {
    if (!hasChanges || !appId) {
      return;
    }
    updateMutation.mutate({ appId, defaultBranch: effectiveBranch });
  };

  return (
    <SettingCard
      className="px-4 py-[18px]"
      icon={<CodeBranch className="text-gray-12" iconSize="xl-regular" />}
      title="Production branch"
      description="Branch that triggers production deployments"
      contentWidth="w-full lg:w-[320px] justify-end"
      expandable={
        <div className="px-4 py-4 flex flex-col gap-4 bg-grayA-2 rounded-b-xl">
          <span className="text-gray-9 text-[13px]">
            Pushes to this branch deploy to the production environment. All other branches deploy to
            preview.
          </span>
          <Combobox
            options={branchOptions}
            value={effectiveBranch}
            onSelect={setSelectedBranch}
            creatable
            searchPlaceholder="Search or type a branch name..."
            emptyMessage="No branches found."
            className="h-8 border-grayA-4 bg-transparent text-[13px] font-medium shadow-md"
            wrapperClassName="w-[240px]"
          />
          <div className="flex justify-end pt-1">
            <InfoTooltip
              content={!hasChanges ? "No changes to save" : undefined}
              disabled={hasChanges}
              asChild
              variant="inverted"
            >
              <Button
                variant="primary"
                className="px-3 py-3"
                size="sm"
                disabled={!hasChanges}
                loading={updateMutation.isLoading}
                onClick={handleSave}
              >
                Save
              </Button>
            </InfoTooltip>
          </div>
        </div>
      }
      chevronState="interactive"
    >
      <SelectedConfig label={currentDefaultBranch} />
    </SettingCard>
  );
};
