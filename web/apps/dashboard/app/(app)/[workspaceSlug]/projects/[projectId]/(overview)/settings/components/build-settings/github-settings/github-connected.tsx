import { trpc } from "@/lib/trpc/client";
import { Button, InfoTooltip, toast } from "@unkey/ui";
import { SelectedConfig } from "../../shared/selected-config";
import { GitHubSettingCard, ManageGitHubAppLink, RepoNameLabel } from "./shared";

export const GitHubConnected = ({
  appId,
  installUrl,
  repoFullName,
}: {
  appId: string;
  installUrl: string;
  repoFullName: string;
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

  const collapsed = (
    <InfoTooltip
      content="Connected repository. Expand to disconnect or manage settings."
      variant="inverted"
      position={{
        side: "top",
      }}
    >
      <SelectedConfig label={<RepoNameLabel fullName={repoFullName} />} />
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
            onClick={() => disconnectRepoMutation.mutate({ appId })}
            loading={disconnectRepoMutation.isLoading}
          >
            Disconnect
          </Button>
        </div>
        <ManageGitHubAppLink
          installUrl={installUrl}
          variant="outline"
          text={<span>Manage GitHub</span>}
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
