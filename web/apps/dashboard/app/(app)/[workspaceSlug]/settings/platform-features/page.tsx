"use client";

import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import {
  type WorkspaceFlag,
  listWorkspaceFlags,
  removeWorkspaceFlagOverride,
  setWorkspaceFlagOverride,
} from "@/lib/workspace-flags-api";
import { useWorkspace } from "@/providers/workspace-provider";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import {
  Button,
  PageBody,
  PageContainer,
  PageHeader,
  PageHeaderContent,
  PageHeaderTitle,
  Skeleton,
  toast,
} from "@unkey/ui";
import { FlagsPanel } from "./flags-panel";

export default function FlagsPage() {
  const workspace = useWorkspaceNavigation();
  const { user } = useWorkspace();
  const queryClient = useQueryClient();
  const queryKey = ["workspace-flags", workspace.id];
  const query = useQuery({
    queryKey,
    queryFn: ({ signal }) => listWorkspaceFlags(signal),
    enabled: Boolean(workspace.id),
    staleTime: 30_000,
    refetchOnWindowFocus: false,
  });

  async function update(operation: () => Promise<WorkspaceFlag>) {
    await queryClient.cancelQueries({ queryKey });
    const flag = await operation();
    await queryClient.cancelQueries({ queryKey });
    queryClient.setQueryData<WorkspaceFlag[]>(queryKey, (current) =>
      current?.map((item) => (item.slug === flag.slug ? flag : item)),
    );
    toast.success("Platform feature updated");
  }

  return (
    <PageContainer>
      <PageHeader>
        <PageHeaderContent>
          <PageHeaderTitle>Platform features</PageHeaderTitle>
        </PageHeaderContent>
      </PageHeader>
      <PageBody>
        <p className="text-sm text-gray-11">
          Enable or disable platform features for everyone in this workspace, or use Unkey's
          defaults.
        </p>
        {query.isLoading ? (
          <output aria-label="Loading platform features">
            <Skeleton className="h-40 w-full rounded-lg" />
          </output>
        ) : query.isError ? (
          <div role="alert" className="rounded-lg border border-grayA-4 p-6">
            <p className="mb-3 text-sm text-gray-11">
              We couldn't load platform features. Try again.
            </p>
            <Button variant="outline" onClick={() => void query.refetch()}>
              Retry
            </Button>
          </div>
        ) : (
          <FlagsPanel
            key={workspace.id}
            flags={query.data ?? []}
            isAdmin={user?.role === "admin"}
            onSet={(slug, value) => update(() => setWorkspaceFlagOverride(slug, value))}
            onRemove={(slug) => update(() => removeWorkspaceFlagOverride(slug))}
          />
        )}
      </PageBody>
    </PageContainer>
  );
}
