import { Combobox } from "@/components/ui/combobox";
import { trpc } from "@/lib/trpc/client";
import { Github, Magnifier } from "@unkey/icons";
import { Input, toast } from "@unkey/ui";
import { useMemo, useState } from "react";
import { RepoListItem } from "./repo-list-item";
import { SelectRepoSkeleton } from "./skeleton";

export const SelectRepo = ({
  projectId,
}: {
  projectId: string;
}) => {
  const utils = trpc.useUtils();
  const [selectedOwner, setSelectedOwner] = useState("");
  const [searchQuery, setSearchQuery] = useState("");

  const { data: reposData, isLoading: isLoadingRepos } = trpc.github.listRepositories.useQuery(
    {
      projectId,
    },
    {
      refetchOnWindowFocus: false,
    },
  );

  const selectRepoMutation = trpc.github.selectRepository.useMutation({
    onSuccess: async () => {
      toast.success("Repository connected");
      await utils.github.getInstallations.invalidate();
    },
    onError: (error) => {
      toast.error(error.message);
    },
  });

  const ownerOptions = useMemo(() => {
    const owners = new Set(
      (reposData?.repositories ?? []).map((repo) => repo.fullName.split("/")[0]),
    );
    return [...owners].map((owner) => ({
      value: owner,
      label: <span className="text-[13px] text-gray-12 font-medium">{owner}</span>,
      searchValue: owner,
      selectedLabel: <span className="text-[13px] text-gray-12 font-medium">{owner}</span>,
    }));
  }, [reposData?.repositories]);

  const filteredRepos = useMemo(
    () =>
      (reposData?.repositories ?? []).filter((repo) => {
        const matchesOwner = !selectedOwner || repo.fullName.startsWith(`${selectedOwner}/`);
        const matchesSearch =
          !searchQuery || repo.fullName.toLowerCase().includes(searchQuery.toLowerCase());
        return matchesOwner && matchesSearch;
      }),
    [reposData?.repositories, selectedOwner, searchQuery],
  );

  const handleSelectOwner = (value: string) => {
    setSelectedOwner(value);
    setSearchQuery("");
  };

  const handleSelectRepository = (repo: {
    id: number;
    fullName: string;
    installationId: number;
  }) => {
    selectRepoMutation.mutate({
      projectId,
      repositoryId: repo.id,
      repositoryFullName: repo.fullName,
      installationId: repo.installationId,
    });
  };

  return (
    <div onClick={(e) => e.stopPropagation()} onKeyDown={(e) => e.stopPropagation()}>
      <div className="flex gap-2 w-full">
        {isLoadingRepos ? (
          <SelectRepoSkeleton />
        ) : ownerOptions.length ? (
          <div className="flex gap-2 w-full pt-1">
            <Combobox
              wrapperClassName="w-[200px] shrink-0"
              className="w-[200px] shrink-0 text-left h-9 border-grayA-4 bg-transparent [&_svg]:text-gray-12"
              options={ownerOptions}
              value={selectedOwner}
              onSelect={handleSelectOwner}
              placeholder={<span className="text-left w-full">Select an account...</span>}
              searchPlaceholder="Filter accounts..."
              leftIcon={<Github />}
            />
            <Input
              className="flex-1 min-w-0 bg-transparent h-9 border-grayA-4"
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              placeholder="Search repositories..."
              leftIcon={<Magnifier iconSize="sm-regular" className="text-gray-12 shrink-0 size-3" />}
            />
          </div>) : null}
      </div>

      {(reposData?.repositories ?? []).length > 0 && (
        filteredRepos.length > 0 ? (
          <ul className="mt-3 flex flex-col border rounded-[14px] border-grayA-5 divide-y divide-grayA-5 min-w-[640px]">
            {filteredRepos.map((repo) => (
              <li key={repo.id} className="animate-in fade-in duration-300">
                <RepoListItem
                  repo={repo}
                  projectId={projectId}
                  onSelect={handleSelectRepository}
                  disabled={selectRepoMutation.isLoading}
                />
              </li>
            ))}
          </ul>
        ) : (
          <div className="mt-3 flex items-center justify-center min-w-[640px] h-[200px] border border-dashed rounded-[14px] border-grayA-5">
            <p className="text-sm text-gray-9">No repositories found</p>
          </div>
        )
      )}
    </div>
  );
};
