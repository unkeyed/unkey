"use client";

import { trpc } from "@/lib/trpc/client";
import { Github, InputSearch } from "@unkey/icons";
import { Button, Empty, Input, Loading, toast } from "@unkey/ui";
import { useParams, useSearchParams } from "next/navigation";
import { useMemo, useState } from "react";
import { useProject } from "../layout-provider";

export default function GitHubPage() {
  const { projectId } = useProject();
  const params = useParams<{ workspaceSlug: string }>();
  const searchParams = useSearchParams();
  const justInstalled = searchParams?.get("installed") === "true";

  const [selectedRepo, setSelectedRepo] = useState<string | null>(null);
  const [searchQuery, setSearchQuery] = useState("");

  const utils = trpc.useUtils();

  const {
    data: installationData,
    isLoading: isLoadingInstallation,
    refetch: refetchInstallation,
  } = trpc.github.getInstallation.useQuery(
    { projectId },
    {
      staleTime: 0,
      refetchOnWindowFocus: true,
    },
  );

  const { data: reposData, isLoading: isLoadingRepos } = trpc.github.listRepositories.useQuery(
    { projectId },
    {
      enabled:
        !!installationData?.installation && !installationData.installation.repositoryFullName,
    },
  );

  const filteredRepos = useMemo(() => {
    if (!reposData?.repositories) {
      return [];
    }
    if (!searchQuery.trim()) {
      return reposData.repositories;
    }
    const query = searchQuery.toLowerCase();
    return reposData.repositories.filter((repo) => repo.fullName.toLowerCase().includes(query));
  }, [reposData?.repositories, searchQuery]);

  const selectRepoMutation = trpc.github.selectRepository.useMutation({
    onSuccess: async () => {
      toast.success("Repository connected");
      await utils.github.getInstallation.invalidate();
      await refetchInstallation();
    },
    onError: (error) => {
      toast.error(error.message);
    },
  });

  const disconnectMutation = trpc.github.disconnect.useMutation({
    onSuccess: async () => {
      toast.success("GitHub disconnected");
      await utils.github.getInstallation.invalidate();
      await refetchInstallation();
      setSelectedRepo(null);
    },
    onError: (error) => {
      toast.error(error.message);
    },
  });

  const handleConnectGitHub = () => {
    const state = `${projectId}:${params?.workspaceSlug ?? ""}`;
    const installUrl = `https://github.com/apps/${process.env.NEXT_PUBLIC_GITHUB_APP_NAME}/installations/new?state=${encodeURIComponent(state)}`;
    window.location.href = installUrl;
  };

  const handleSelectRepository = () => {
    if (!selectedRepo) {
      return;
    }
    const repo = reposData?.repositories.find((r) => r.fullName === selectedRepo);
    if (!repo) {
      return;
    }
    selectRepoMutation.mutate({
      projectId,
      repositoryId: repo.id,
    });
  };

  const handleDisconnect = () => {
    disconnectMutation.mutate({ projectId });
  };

  if (isLoadingInstallation) {
    return (
      <div className="flex items-center justify-center h-full">
        <Loading />
      </div>
    );
  }

  const installation = installationData?.installation;

  if (installation?.repositoryFullName) {
    return (
      <div className="flex flex-col items-center justify-center h-full p-8">
        <Empty>
          <Empty.Icon>
            <Github className="text-success-9" />
          </Empty.Icon>
          <Empty.Title>Connected to GitHub</Empty.Title>
          <Empty.Description>
            <a
              href={`https://github.com/${installation.repositoryFullName}`}
              target="_blank"
              rel="noopener noreferrer"
              className="text-accent-11 hover:underline font-medium"
            >
              {installation.repositoryFullName}
            </a>
          </Empty.Description>
          <Empty.Actions>
            <Button
              variant="outline"
              color="danger"
              onClick={handleDisconnect}
              loading={disconnectMutation.isLoading}
            >
              Disconnect
            </Button>
          </Empty.Actions>
        </Empty>
      </div>
    );
  }

  if (installation || justInstalled) {
    return (
      <div className="flex flex-col items-center justify-center h-full p-8">
        <Empty>
          <Empty.Icon>
            <Github className="text-accent-11" />
          </Empty.Icon>
          <Empty.Title>Select a Repository</Empty.Title>
          <Empty.Description>Choose which repository to connect to this project.</Empty.Description>
          <div className="mt-4 w-full max-w-md">
            {isLoadingRepos ? (
              <Loading />
            ) : reposData?.repositories.length ? (
              <div className="space-y-3">
                <Input
                  placeholder="Search repositories..."
                  value={searchQuery}
                  onChange={(e) => setSearchQuery(e.target.value)}
                  leftIcon={<InputSearch className="size-4 text-gray-9" />}
                />
                <div className="border border-gray-6 rounded-lg h-[50vh] overflow-y-auto">
                  {filteredRepos.length > 0 ? (
                    filteredRepos.map((repo) => (
                      <button
                        type="button"
                        key={repo.id}
                        onClick={() => setSelectedRepo(repo.fullName)}
                        className={`w-full px-3 py-2 text-left text-sm flex items-center justify-between hover:bg-gray-3 transition-colors border-b border-gray-6 last:border-b-0 ${
                          selectedRepo === repo.fullName ? "bg-gray-4" : ""
                        }`}
                      >
                        <span className="text-gray-12">{repo.fullName}</span>
                        {repo.private && <span className="text-xs text-gray-9 ml-2">private</span>}
                      </button>
                    ))
                  ) : (
                    <div className="py-4 px-3 text-gray-9 text-sm text-center">
                      No repositories match your search
                    </div>
                  )}
                </div>
                <Button
                  className="w-full"
                  onClick={handleSelectRepository}
                  disabled={!selectedRepo}
                  loading={selectRepoMutation.isLoading}
                >
                  Connect Repository
                </Button>
              </div>
            ) : (
              <p className="text-gray-9 text-sm">
                No repositories found. Please check your GitHub App permissions.
              </p>
            )}
          </div>
        </Empty>
      </div>
    );
  }

  return (
    <div className="flex flex-col items-center justify-center h-full p-8">
      <Empty>
        <Empty.Icon>
          <Github className="text-accent-11" />
        </Empty.Icon>
        <Empty.Title>Connect GitHub Repository</Empty.Title>
        <Empty.Description>
          Connect a GitHub repository to enable automatic deployments on push.
        </Empty.Description>
        <Empty.Actions>
          <Button onClick={handleConnectGitHub}>
            <Github className="size-4" />
            Connect GitHub
          </Button>
        </Empty.Actions>
      </Empty>
    </div>
  );
}
