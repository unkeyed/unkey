import { Github } from "@unkey/icons";
import { Button, type ChevronState, SettingCard } from "@unkey/ui";

export const GitHubSettingCard = ({
  children,
  expandable,
  chevronState,
}: {
  children: React.ReactNode;
  expandable?: React.ReactNode;
  chevronState: ChevronState;
}) => (
  <SettingCard
    className="px-4 py-[18px]"
    icon={<Github className="text-gray-12" iconSize="xl-regular" />}
    title="Repository"
    description="Source repository for this deployment"
    border="both"
    contentWidth="w-full lg:w-[320px] justify-end"
    expandable={expandable}
    chevronState={chevronState}
  >
    {children}
  </SettingCard>
);

export const ComboboxSkeleton = () => (
  <div className="w-[250px] h-8 rounded-lg border border-gray-5 bg-gray-2 flex items-center justify-between px-3 py-2">
    <div className="flex gap-1.5 items-center">
      <div className="h-3.5 w-16 bg-grayA-3 rounded animate-pulse" />
      <div className="h-3.5 w-24 bg-grayA-3 rounded animate-pulse" />
    </div>
    <div className="h-4 w-4 bg-grayA-3 rounded animate-pulse" />
  </div>
);

export const RepoNameLabel = ({ fullName }: { fullName: string }) => {
  const [handle, repoName] = fullName.split("/");
  return (
    <div>
      <span className="text-[13px] text-gray-12 font-medium">{handle}</span>
      <span className="text-[13px] text-gray-11">/{repoName}</span>
    </div>
  );
};

export const ManageGitHubAppLink = ({
  installUrl,
  variant = "ghost",
  className = "-ml-3 px-3 py-2 rounded-lg",
  text = "Manage Github App",
}: {
  installUrl: string;
  variant?: "outline" | "ghost";
  className?: string;
  text?: React.ReactNode;
}) => (
  <Button variant={variant} className={className}>
    <a href={installUrl} className="text-sm text-gray-12" target="_blank" rel="noopener noreferrer">
      {text}
    </a>
  </Button>
);
