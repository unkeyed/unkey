import { Combobox } from "@/components/ui/combobox";
import { trpc } from "@/lib/trpc/client";
import { Button, InfoTooltip, toast } from "@unkey/ui";
import { GitHubSettingCard, ManageGitHubAppLink, RepoNameLabel } from "./shared";

export const GitHubConnected = ({
  projectId,
  installUrl,
  repoFullName,
  repositoryId,
}: {
  projectId: string;
  installUrl: string;
  repoFullName: string;
  repositoryId: number;
}) => {
  const utils = trpc.useUtils();

  const disconnectRepoMutation = trpc.github.disconnectRepo.useMutation({
    onSuccess: async () => {
      toast.success("Repository disconnected");
      await utils.github.getInstallations.invalidate();
    },
    onError: (error) => {
      toast.error(error.message);
    },
  });

  const connectedValue = String(repositoryId);
  const connectedOption = [
    {
      value: connectedValue,
      label: repoFullName,
      searchValue: repoFullName,
      selectedLabel: <RepoNameLabel fullName={repoFullName} />,
    },
  ];

  const collapsed = (
    <InfoTooltip
      asChild
      className="pointer-events-none"
      content="Connected repository. Expand to disconnect or manage settings."
      variant="inverted"
      position={{
        side: "top",
      }}
    >
      {/* Without this wrapper div combobox will trigger expandable content */}
      <div onClick={(e) => e.stopPropagation()} onKeyDown={(e) => e.stopPropagation()}>
        <Combobox
          className="w-[250px] text-left min-h-8 pointer-events-none opacity-75"
          options={connectedOption}
          value={connectedValue}
          onSelect={() => {}}
        />
      </div>
    </InfoTooltip>
  );

  const expandable = (
    <div className="px-6 py-4 flex flex-col gap-3 bg-grayA-2 rounded-b-xl">
      <span className="text-gray-9 text-[13px]">
        Pushes to this repository will trigger deployments.
      </span>
      <div className="flex items-center gap-5 pt-1">
        <div onClick={(e) => e.stopPropagation()} onKeyDown={(e) => e.stopPropagation()}>
          <Button
            className="px-3 rounded-lg"
            variant="primary"
            color="danger"
            onClick={() => disconnectRepoMutation.mutate({ projectId })}
            loading={disconnectRepoMutation.isLoading}
          >
            Disconnect
          </Button>
        </div>
        <ManageGitHubAppLink
          installUrl={installUrl}
          text={
            <>
              <span className="text-gray-9">Manage</span>
              <span className="text-gray-12 font-medium"> GitHub</span>
            </>
          }
        />
      </div>
    </div>
  );

  return (
    <GitHubSettingCard expandable={expandable} chevronState="interactive">
      {collapsed}
    </GitHubSettingCard>
  );
};
