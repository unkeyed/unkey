import { Button, SettingsRow, Skeleton } from "@unkey/ui";

export const GitHubSettingCard = ({ children }: { children: React.ReactNode }) => (
  <SettingsRow title="Repository" description="Source repository for this deployment">
    {children}
  </SettingsRow>
);

export const ComboboxSkeleton = () => (
  <div className="w-[185px] h-7 rounded-lg border bg-raised flex items-center justify-between px-3 py-2">
    <div className="flex gap-1.5 items-center">
      <Skeleton className="h-3.5 w-16 rounded" />
      <Skeleton className="h-3.5 w-24 rounded" />
    </div>
    <Skeleton className="h-4 w-4 rounded" />
  </div>
);

export const RepoNameLabel = ({ fullName }: { fullName: string }) => {
  const [handle, repoName] = fullName.split("/");
  return (
    // This max-w-[185px] and w-[185px] in ComboboxSkeleton should match
    <div className="max-w-[185px] truncate">
      <span className="text-sm text-gray-12 font-medium">{handle}</span>
      <span className="text-sm text-gray-11">/{repoName}</span>
    </div>
  );
};

export const ManageGitHubAppLink = ({
  onInstall,
  variant = "primary",
  className = "px-3 py-2 rounded-md",
  text = "Manage Github App",
}: {
  onInstall: () => Promise<void> | void;
  variant?: "outline" | "ghost" | "primary";
  className?: string;
  text?: React.ReactNode;
}) => (
  <Button
    variant={variant}
    className={className}
    onClick={(e) => {
      e.preventDefault();
      void onInstall();
    }}
  >
    <span className="text-sm">{text}</span>
  </Button>
);
