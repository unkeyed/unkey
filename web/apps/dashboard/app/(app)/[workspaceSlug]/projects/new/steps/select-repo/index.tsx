import { Combobox } from "@/components/ui/combobox";
import { trpc } from "@/lib/trpc/client";
import { useVirtualizer } from "@tanstack/react-virtual";
import { Check, Github, Magnifier, XMark } from "@unkey/icons";
import { Button, Input, toast, useStepWizard } from "@unkey/ui";
import { useMemo, useRef, useState } from "react";
import { RepoListItem } from "./repo-list-item";
import { SelectRepoSkeleton } from "./skeleton";

export const SelectRepo = ({
  projectId,
}: {
  projectId: string;
}) => {
  const { next } = useStepWizard();
  const trpcUtils = trpc.useUtils();
  const [selectedOwner, setSelectedOwner] = useState("");
  const [searchQuery, setSearchQuery] = useState("");
  const [isBannerDismissed, setIsBannerDismissed] = useState(false);
  const [mutatingRepoId, setMutatingRepoId] = useState<number | null>(null);

  const {
    data: reposData,
    isLoading: isLoadingRepos,
    error: reposError,
  } = trpc.github.listRepositories.useQuery(
    {
      projectId,
    },
    {
      refetchOnWindowFocus: false,
    },
  );

  const selectRepoMutation = trpc.github.selectRepository.useMutation({
    onSuccess: async (_data, variables) => {
      trpcUtils.github.getInstallations.invalidate();
      const name =
        variables.repositoryFullName.length > 40
          ? `${variables.repositoryFullName.slice(0, 37)}...`
          : variables.repositoryFullName;
      toast.success(
        <span className="text-gray-11">
          <span className="text-gray-12 font-medium">{name} </span>linked
        </span>,
      );
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

  const parentRef = useRef<HTMLDivElement>(null);
  const virtualizer = useVirtualizer({
    count: filteredRepos.length,
    getScrollElement: () => parentRef.current,
    estimateSize: () => 77,
    overscan: 3,
  });

  const handleSelectOwner = (value: string) => {
    setSelectedOwner(value);
    setSearchQuery("");
  };

  const handleSelectRepository = async (repo: {
    id: number;
    fullName: string;
    installationId: number;
  }) => {
    setMutatingRepoId(repo.id);
    try {
      await selectRepoMutation.mutateAsync({
        projectId,
        repositoryId: repo.id,
        repositoryFullName: repo.fullName,
        installationId: repo.installationId,
      });
      next();
    } finally {
      setMutatingRepoId(null);
    }
  };

  return (
    <div onClick={(e) => e.stopPropagation()} onKeyDown={(e) => e.stopPropagation()}>
      {!isBannerDismissed && (
        <div className="absolute top-2 left-2 right-2 z-50 rounded-[10px] p-3 gap-2.5 flex items-center shadow-[inset_0_0_0_0.75px_rgba(0,0,0,0.10)] bg-linear-to-r from-successA-4 via-successA-1 to-success-1">
          <Check iconSize="sm-regular" />
          <div className="flex items-center gap-1">
            <span className="font-medium text-[13px] text-success-12">
              GitHub connected successfully.
            </span>
            <span className="text-[13px] text-success-12">
              You can now select a repository to deploy
            </span>
          </div>
          <button type="button" onClick={() => setIsBannerDismissed(true)} className="ml-auto">
            <XMark iconSize="sm-regular" />
          </button>
        </div>
      )}

      <div className="flex gap-2 w-full min-w-[600px]">
        {isLoadingRepos ? (
          <SelectRepoSkeleton />
        ) : reposError ? (
          <div className="mt-3 flex flex-col items-center justify-center min-w-[640px] h-[462px] gap-3 border border-dashed rounded-[14px] border-grayA-5">
            <p className="text-[15px] text-accent-12 font-semibold">Failed to load repositories</p>
            <p className="text-[13px] text-accent-11 text-center whitespace-pre-line w-[350px]">
              {reposError.message}
            </p>
            <Button
              variant="primary"
              className="px-3"
              size="sm"
              onClick={() => trpcUtils.github.listRepositories.invalidate()}
            >
              Retry
            </Button>
          </div>
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
              leftIcon={
                <Magnifier iconSize="sm-regular" className="text-gray-12 shrink-0 size-3" />
              }
            />
          </div>
        ) : null}
      </div>

      {(reposData?.repositories ?? []).length > 0 &&
        (filteredRepos.length > 0 ? (
          <div
            ref={parentRef}
            className="mt-3 border rounded-[14px] border-grayA-5 min-w-[640px] max-h-[462px] overflow-y-auto"
          >
            <div style={{ height: `${virtualizer.getTotalSize()}px`, position: "relative" }}>
              {virtualizer.getVirtualItems().map((virtualRow) => {
                const repo = filteredRepos[virtualRow.index];
                return (
                  <div
                    key={repo.id}
                    ref={virtualizer.measureElement}
                    data-index={virtualRow.index}
                    className={
                      virtualRow.index < filteredRepos.length - 1 ? "border-b border-grayA-5" : ""
                    }
                    style={{
                      position: "absolute",
                      top: 0,
                      left: 0,
                      width: "100%",
                      transform: `translateY(${virtualRow.start}px)`,
                    }}
                  >
                    <RepoListItem
                      repo={repo}
                      projectId={projectId}
                      onSelect={handleSelectRepository}
                      disabled={mutatingRepoId !== null}
                      loading={mutatingRepoId === repo.id}
                    />
                  </div>
                );
              })}
            </div>
          </div>
        ) : (
          <div className="mt-3 flex flex-col items-center justify-center min-w-[640px] h-[462px] gap-3 border border-dashed rounded-[14px] border-grayA-5">
            <p className="text-[15px] text-accent-12 font-semibold">No repositories found</p>
          </div>
        ))}
    </div>
  );
};
