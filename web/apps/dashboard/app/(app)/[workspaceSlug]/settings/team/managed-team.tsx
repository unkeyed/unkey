"use client";

import "@unkey/workos-widgets/styles.css";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { logManagedAuthOutcome } from "@/lib/auth/telemetry";
import { routes } from "@/lib/navigation/routes";
import { Button, Empty, Skeleton } from "@unkey/ui";
import { ManagedUsersWidget } from "@unkey/workos-widgets";
import { useAccessToken, useAuth } from "@workos-inc/authkit-nextjs/components";
import Link from "next/link";
import { useCallback, useState } from "react";

const MANAGE_USERS_PERMISSION = "widgets:users-table:manage";

export function ManagedTeam({ team }: { team: boolean }) {
  const workspace = useWorkspaceNavigation();
  const { user, impersonator, permissions, loading } = useAuth();

  if (!team) {
    return (
      <div className="flex min-h-[60vh] w-full items-center justify-center">
        <Empty className="w-full">
          <Empty.Title>Upgrade Your Plan to Add Team Members</Empty.Title>
          <Empty.Description>You can try it out for free for 14 days.</Empty.Description>
          <Empty.Actions>
            <Button
              render={<Link href={routes.settings.billing({ workspaceSlug: workspace.slug })} />}
            >
              Upgrade
            </Button>
          </Empty.Actions>
        </Empty>
      </div>
    );
  }

  if (loading) {
    return <ManagedTeamSkeleton />;
  }

  if (!user) {
    return (
      <ManagedTeamError
        heading="Your session has expired"
        description="Sign in again to manage workspace members."
        action={
          <Link className="underline" href={routes.auth.signIn()}>
            Sign in again
          </Link>
        }
      />
    );
  }

  if (impersonator) {
    return (
      <ManagedTeamError
        heading="Team management unavailable"
        description="Managed team controls are disabled while you are impersonating another user."
      />
    );
  }

  if (!permissions?.includes(MANAGE_USERS_PERMISSION)) {
    return (
      <ManagedTeamError
        heading="Admin access required"
        description="Your WorkOS role does not allow you to manage workspace members."
      />
    );
  }

  return <ManagedTeamWidgets />;
}

function ManagedTeamWidgets() {
  const { accessToken, loading, error, refresh, getAccessToken } = useAccessToken();
  const [retryError, setRetryError] = useState<string | null>(null);
  const [retrying, setRetrying] = useState(false);

  const getWidgetAccessToken = useCallback(async () => {
    try {
      const token = await getAccessToken();
      if (!token) {
        throw new Error("Session expired");
      }
      logManagedAuthOutcome("widget_token", "success");
      return token;
    } catch {
      logManagedAuthOutcome("widget_token", "failure");
      throw new Error("Your session has expired.");
    }
  }, [getAccessToken]);

  if (loading || (!accessToken && !error && !retryError)) {
    return <ManagedTeamSkeleton />;
  }

  if (error || retryError) {
    return (
      <ManagedTeamError
        heading="Team management is temporarily unavailable"
        description={retryError ?? "WorkOS could not load workspace members."}
        action={
          <Button
            type="button"
            disabled={retrying}
            loading={retrying}
            onClick={async () => {
              setRetrying(true);
              setRetryError(null);
              try {
                const token = await refresh();
                if (!token) {
                  throw new Error("Session expired");
                }
                logManagedAuthOutcome("widget_token", "success");
              } catch {
                logManagedAuthOutcome("widget_token", "failure");
                setRetryError("We could not refresh your account session. Sign in again.");
              } finally {
                setRetrying(false);
              }
            }}
          >
            Retry
          </Button>
        }
      />
    );
  }

  return (
    <section aria-labelledby="members-heading" className="flex flex-col gap-3">
      <div className="flex flex-col gap-1">
        <h2 id="members-heading" className="m-0 text-lg font-medium">
          Members
        </h2>
        <p className="m-0 text-sm text-gray-11">Manage workspace members and invitations.</p>
      </div>
      <ManagedUsersWidget getAccessToken={getWidgetAccessToken} />
    </section>
  );
}

function ManagedTeamSkeleton() {
  return (
    <section
      aria-busy="true"
      aria-labelledby="members-loading-heading"
      className="flex flex-col gap-3"
    >
      <output aria-live="polite" className="sr-only">
        Loading workspace members...
      </output>
      <div className="flex flex-col gap-1">
        <h2 id="members-loading-heading" className="m-0 text-lg font-medium">
          Members
        </h2>
        <p className="m-0 text-sm text-gray-11">Manage workspace members and invitations.</p>
      </div>
      <div aria-hidden="true" className="flex flex-col gap-3">
        <div className="flex gap-2">
          <Skeleton className="h-8 w-80 max-w-full" />
          <Skeleton className="ml-auto h-8 w-28 shrink-0" />
        </div>
        <div className="overflow-hidden rounded-lg border border-grayA-4">
          <div className="flex min-h-10 items-center gap-4 border-grayA-4 border-b px-4">
            <Skeleton className="h-3 w-40" />
            <Skeleton className="ml-auto h-3 w-20" />
          </div>
          {[0, 1, 2].map((row) => (
            <div
              key={row}
              className="flex min-h-16 items-center gap-3 border-grayA-4 border-b px-4 last:border-b-0"
            >
              <Skeleton className="size-8 shrink-0 rounded-full" />
              <div className="flex flex-1 flex-col gap-2">
                <Skeleton className="h-3.5 w-40 max-w-full" />
                <Skeleton className="h-3 w-56 max-w-full" />
              </div>
              <Skeleton className="h-7 w-20 shrink-0" />
            </div>
          ))}
        </div>
      </div>
    </section>
  );
}

function ManagedTeamError({
  heading,
  description,
  action,
}: {
  heading: string;
  description: string;
  action?: React.ReactNode;
}) {
  return (
    <section className="rounded-lg border border-grayA-4 p-6" role="alert">
      <h2 className="m-0 font-medium">{heading}</h2>
      <p className="mt-2 mb-0 text-sm text-gray-11">{description}</p>
      {action ? <div className="mt-4 text-sm font-medium">{action}</div> : null}
    </section>
  );
}
