import { Combobox } from "@/components/ui/combobox";
import { trpc } from "@/lib/trpc/client";
import { useVirtualizer } from "@tanstack/react-virtual";
import { Check, Clock, Github, Magnifier, XMark } from "@unkey/icons";
import { Button, Input, toast, useStepWizard } from "@unkey/ui";
import { useMemo, useRef, useState } from "react";
import { OnboardingLinks } from "../../onboarding-links";
import { OnboardingStepHint, OnboardingStepHintHighlight } from "../../onboarding-step-hint";
import { RepoListItem } from "./repo-list-item";
import { SelectRepoSkeleton } from "./skeleton";

export const SelectRepo = ({
  projectId,
  onBeforeNavigate,
  hasGithubInstallation,
  onSkip,
}: {
  projectId: string;
  onBeforeNavigate?: () => void;
  hasGithubInstallation: boolean;
  onSkip?: () => void;
}) => {
  const { next } = useStepWizard();
  const trpcUtils = trpc.useUtils();
  const installUrl = `https://github.com/apps/${process.env.NEXT_PUBLIC_GITHUB_APP_NAME}/installations/new?state=${encodeURIComponent(JSON.stringify({ projectId }))}`;

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
      trpcUtils.github.getRepoTree.invalidate();
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

  const handleSelectRepository = async (
    repo: {
      id: number;
      fullName: string;
      installationId: number;
    },
    selectedBranch: string,
  ) => {
    setMutatingRepoId(repo.id);
    try {
      await selectRepoMutation.mutateAsync({
        projectId,
        repositoryId: repo.id,
        repositoryFullName: repo.fullName,
        installationId: repo.installationId,
        selectedBranch,
      });
      next();
    } finally {
      setMutatingRepoId(null);
    }
  };

  return (
    <div
      onClick={(e) => e.stopPropagation()}
      onKeyDown={(e) => e.stopPropagation()}
      className="[--repo-list-w:750px]"
    >
      {!isBannerDismissed && (
        <div className="absolute top-2 left-2 right-2 z-50 rounded-[10px] p-3 gap-2.5 flex items-center shadow-[inset_0_0_0_0.75px_rgba(0,0,0,0.10)] bg-linear-to-r from-successA-4 via-successA-1 to-success-1">
          <Check iconSize="sm-regular" className="text-successA-12" />
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

      <div className="flex gap-2 min-w-[var(--repo-list-w)]">
        {isLoadingRepos ? (
          <SelectRepoSkeleton />
        ) : reposError ? (
          <div className="mt-3 flex flex-col items-center justify-center min-w-[var(--repo-list-w)] h-[462px] gap-3 border border-dashed rounded-[14px] border-grayA-5">
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
          <div className="flex gap-2 min-w-[var(--repo-list-w)] pt-1">
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
            className="mt-3 border rounded-[14px] border-grayA-5 min-w-[var(--repo-list-w)] max-h-[462px] overflow-y-auto"
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
                      hasMultipleAccounts={ownerOptions.length > 1}
                    />
                  </div>
                );
              })}
            </div>
          </div>
        ) : (
          <div className="mt-3 flex flex-col items-center justify-center min-w-[var(--repo-list-w)] h-[462px] gap-3 border border-dashed rounded-[14px] border-grayA-5">
            <p className="text-[15px] text-accent-12 font-semibold">No repositories found</p>
          </div>
        ))}

      {onSkip && (
        <div className="mt-3 border border-grayA-5 rounded-[14px] flex justify-start items-center gap-4 py-[18px] px-4 min-w-[var(--repo-list-w)]">
          <div className="size-8 rounded-[10px] grid place-items-center ring-1 ring-grayA-4 shadow-sm shadow-grayA-8/20 dark:shadow-none">
            <Clock className="size-[18px] text-gray-12" iconSize="md-medium" />
          </div>
          <div className="flex flex-col gap-3">
            <span className="font-medium text-gray-12 text-[13px] leading-[9px]">
              Skip GitHub setup
            </span>
            <span className="text-gray-10 text-[13px] leading-[9px]">
              Continue without a repository. You can connect GitHub later from project settings.
            </span>
          </div>
          <Button
            variant="outline"
            onClick={onSkip}
            className="ml-auto rounded-lg border-grayA-4 hover:bg-grayA-2 shadow-sm hover:shadow-md transition-all"
          >
            <span className="text-[13px] text-gray-12 font-medium">Skip for now</span>
          </Button>
        </div>
      )}

      {hasGithubInstallation && (
        <a href={installUrl} rel="noopener noreferrer" onClick={onBeforeNavigate} className="group">
          <OnboardingStepHint>
            Can't find your repo? Add more from{" "}
            <OnboardingStepHintHighlight>GitHub</OnboardingStepHintHighlight>.
          </OnboardingStepHint>
        </a>
      )}
      <div className="mt-8 min-w-[var(--repo-list-w)] items-center justify-center flex">
        <OnboardingLinks />
      </div>
    </div>
  );
};
