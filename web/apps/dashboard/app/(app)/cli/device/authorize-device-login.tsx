"use client";

import { buildUrns } from "@/app/(app)/[workspaceSlug]/settings/root-keys/components/builder/lib/urn";
import {
  OnboardingCard,
  OnboardingCardContent,
  OnboardingCardFooter,
} from "@/app/new/components/onboarding-card";
import { LoadError } from "@/components/load-error";
import { Logomark } from "@/components/logomark";
import {
  type DeviceLogin,
  approveDeviceLogin,
  denyDeviceLogin,
  getDeviceLogin,
} from "@/lib/cli/device-api";
import { formatUserCode } from "@/lib/cli/format-user-code";
import { fromQuery } from "@/lib/loadable/loadable";
import { routes } from "@/lib/navigation/routes";
import { trpc } from "@/lib/trpc/client";
import { getErrorMessage } from "@/lib/unkey-client";
import { useWorkspace } from "@/providers/workspace-provider";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
  IconCheckOutline18,
  IconCircleInfoOutline18,
  IconTerminalOutline18,
  IconXmarkOutline18,
} from "@unkey/icons";
import { match } from "@unkey/match";
import { Button, Combobox, FormField, Skeleton, toast } from "@unkey/ui";
import Link from "next/link";
import { type FormEvent, type ReactNode, useEffect, useRef, useState } from "react";
import { type Access, PermissionsField, accessPolicies } from "./permissions-field";

type AuthorizeDeviceLoginProps = {
  userCode: string;
};

export function AuthorizeDeviceLogin({ userCode }: AuthorizeDeviceLoginProps) {
  const { workspace } = useWorkspace();
  const queryClient = useQueryClient();
  const trimmed = userCode.trim();
  const query = useQuery({
    queryKey: ["cli-device", trimmed],
    queryFn: () => getDeviceLogin(trimmed),
    enabled: trimmed.length > 0,
    retry: false,
  });

  const refresh = async (status: "approved" | "denied") => {
    queryClient.setQueryData<DeviceLogin>(["cli-device", trimmed], (login) =>
      login ? { ...login, status } : login,
    );
    await Promise.all([
      queryClient.invalidateQueries({ queryKey: ["cli-device", trimmed] }),
      queryClient.invalidateQueries({ queryKey: ["cli-sessions"] }),
    ]);
  };
  const reload = () => {
    void queryClient.invalidateQueries({ queryKey: ["cli-device", trimmed] });
  };

  const login = fromQuery(query);

  if (trimmed.length === 0) {
    return (
      <CenteredFlow>
        <BrandHeader
          tone="pending"
          title="Sign in to the Unkey CLI"
          description={
            <>
              Run{" "}
              <code className="rounded-md bg-grayA-3 px-1.5 py-0.5 font-mono text-xs text-gray-12">
                unkey login
              </code>{" "}
              and open the link it prints to approve the login.
            </>
          }
        />
      </CenteredFlow>
    );
  }

  return match(login)
    .with({ status: "loading" }, () => (
      <CenteredFlow wide>
        <LoginSkeleton />
      </CenteredFlow>
    ))
    .with({ status: "failed" }, ({ error, retry }) => (
      <CenteredFlow>
        <LoadError title="Could not load login" error={error} frame="none" onRetry={retry} />
      </CenteredFlow>
    ))
    .with({ status: "ready" }, ({ data }) =>
      workspace ? (
        <CenteredFlow wide={data.status === "pending"}>
          <DeviceLoginView
            login={data}
            workspace={workspace}
            onChanged={refresh}
            onStale={reload}
          />
        </CenteredFlow>
      ) : (
        <CenteredFlow wide>
          <LoginSkeleton />
        </CenteredFlow>
      ),
    )
    .exhaustive();
}

function LoginSkeleton() {
  return (
    <div aria-busy="true" className="flex w-full flex-col gap-8">
      <output className="sr-only">Loading login...</output>
      <div aria-hidden="true" className="flex flex-col items-center gap-5">
        <Skeleton className="size-14 rounded-xl" />
        <div className="flex flex-col items-center gap-2">
          <Skeleton className="h-6 w-52" />
          <Skeleton className="h-4 w-72" />
        </div>
      </div>
      <OnboardingCard aria-hidden="true">
        <OnboardingCardContent className="mt-0 flex flex-col gap-6">
          {STEP_SLOTS.map((slot) => (
            <div key={slot} className="flex flex-col gap-2">
              <div className="flex items-center gap-3">
                <Skeleton className="size-6 rounded-full" />
                <Skeleton className="h-4 w-32" />
              </div>
              <div className="flex flex-col gap-2 pl-9">
                <Skeleton className="h-4 w-56" />
                <Skeleton className="h-10 w-full rounded-md" />
              </div>
            </div>
          ))}
        </OnboardingCardContent>
        <OnboardingCardFooter className="flex justify-end gap-2">
          <Skeleton className="h-9 w-16 rounded-md" />
          <Skeleton className="h-9 w-28 rounded-md" />
        </OnboardingCardFooter>
      </OnboardingCard>
    </div>
  );
}

const STEP_SLOTS = ["code", "access"] as const;

function CenteredFlow({ children, wide = false }: { children: ReactNode; wide?: boolean }) {
  return (
    <div className="flex w-full flex-1 flex-col items-center justify-center px-4 py-8 sm:p-12">
      <div
        className={
          wide
            ? "flex w-full max-w-xl flex-col items-center gap-6"
            : "flex w-full max-w-md flex-col items-center gap-6 text-center"
        }
      >
        {children}
      </div>
    </div>
  );
}

type Workspace = { id: string; slug: string };

function DeviceLoginView({
  login,
  workspace,
  onChanged,
  onStale,
}: {
  login: DeviceLogin;
  workspace: Workspace;
  onChanged: (status: "approved" | "denied") => Promise<void>;
  onStale: () => void;
}) {
  return match(login.status)
    .with("pending", () => (
      <PendingLogin
        login={login}
        workspaceId={workspace.id}
        onChanged={onChanged}
        onStale={onStale}
      />
    ))
    .with("approved", "consumed", () => <Approved workspaceSlug={workspace.slug} />)
    .with("denied", () => (
      <BrandHeader
        tone="closed"
        title="This login was denied"
        description="Run unkey login again if you want to start a new one."
      />
    ))
    .otherwise(() => (
      <BrandHeader
        tone="closed"
        title="This login has expired"
        description="Run unkey login again to get a new code."
      />
    ));
}

type BrandTone = "pending" | "success" | "closed";

const BRAND_BADGES: Record<BrandTone, ReactNode> = {
  pending: (
    <span className="absolute -right-2 -bottom-2 flex size-7 items-center justify-center rounded-lg border border-grayA-4 bg-gray-12 text-gray-1">
      <IconTerminalOutline18 className="size-3.5" />
    </span>
  ),
  success: (
    <span className="absolute -right-2 -bottom-2 flex size-7 items-center justify-center rounded-full border-2 border-background bg-success-9 text-white">
      <IconCheckOutline18 className="size-3.5" />
    </span>
  ),
  closed: (
    <span className="absolute -right-2 -bottom-2 flex size-7 items-center justify-center rounded-full border-2 border-background bg-gray-9 text-white">
      <IconXmarkOutline18 className="size-3.5" />
    </span>
  ),
};

function BrandHeader({
  tone,
  title,
  description,
}: {
  tone: BrandTone;
  title: string;
  description: ReactNode;
}) {
  const headingRef = useRef<HTMLHeadingElement>(null);
  useEffect(() => {
    headingRef.current?.focus({ preventScroll: true });
  }, []);

  return (
    <div className="flex flex-col items-center gap-5 text-center">
      <div aria-hidden="true" className="relative">
        <span className="flex size-14 items-center justify-center rounded-xl border border-grayA-4 bg-raised shadow-2xs">
          <Logomark className="size-7 [&>svg]:size-7" />
        </span>
        {BRAND_BADGES[tone]}
      </div>
      <div className="flex flex-col gap-1">
        <h1
          ref={headingRef}
          tabIndex={-1}
          className="text-xl font-semibold text-gray-12 outline-hidden"
        >
          {title}
        </h1>
        <p className="text-sm text-gray-11">{description}</p>
      </div>
    </div>
  );
}

function CodeBoxes({ userCode }: { userCode: string }) {
  const characters = Array.from(formatUserCode(userCode), (char, position) => ({
    char,
    id: `${position}-${char}`,
  }));

  return (
    <div className="flex items-center gap-1 sm:gap-2">
      <span className="sr-only">Login code {formatUserCode(userCode)}</span>
      {characters.map(({ char, id }) => (
        <span
          key={id}
          aria-hidden="true"
          className={
            char === "-"
              ? "font-mono text-base text-gray-11"
              : "flex h-10 w-7 items-center justify-center rounded-md border border-grayA-4 bg-grayA-2 font-mono text-base font-medium text-gray-12 sm:w-10"
          }
        >
          {char}
        </span>
      ))}
    </div>
  );
}

function Step({
  number,
  title,
  description,
  children,
}: {
  number: number;
  title: string;
  description: string;
  children: ReactNode;
}) {
  return (
    <section className="flex flex-col gap-2">
      <div className="flex items-start gap-3">
        <span className="flex size-6 shrink-0 items-center justify-center rounded-full bg-grayA-3 text-xs font-medium text-gray-12">
          {number}
        </span>
        <div className="flex flex-col gap-0.5">
          <h2 className="text-sm font-medium text-gray-12">{title}</h2>
          <p className="text-sm text-gray-11">{description}</p>
        </div>
      </div>
      <div className="pl-9">{children}</div>
    </section>
  );
}

function Approved({ workspaceSlug }: { workspaceSlug: string }) {
  return (
    <div className="flex w-full flex-col items-center gap-8">
      <BrandHeader
        tone="success"
        title="You're signed in to the Unkey CLI"
        description="Head back to your terminal to keep going. You can close this tab."
      />
      <div className="flex items-center gap-3 text-sm text-gray-11">
        <Link
          href={routes.settings.cliSessions({ workspaceSlug })}
          className="underline-offset-4 hover:text-gray-12 hover:underline"
        >
          Manage CLI sessions
        </Link>
        <span aria-hidden="true">·</span>
        <a
          href="https://www.unkey.com/docs/platform/cli/overview"
          target="_blank"
          rel="noopener noreferrer"
          className="underline-offset-4 hover:text-gray-12 hover:underline"
        >
          CLI docs
        </a>
      </div>
    </div>
  );
}

function PendingLogin({
  login,
  workspaceId,
  onChanged,
  onStale,
}: {
  login: DeviceLogin;
  workspaceId: string;
  onChanged: (status: "approved" | "denied") => Promise<void>;
  onStale: () => void;
}) {
  const [access, setAccess] = useState<Access>({ kind: "none" });
  const [missing, setMissing] = useState(false);
  const permissionsRef = useRef<HTMLButtonElement>(null);
  const approve = useMutation({
    mutationFn: approveDeviceLogin,
    async onSuccess() {
      await onChanged("approved");
    },
    onError(error) {
      toast.error(getErrorMessage(error, "Could not approve this login."));
      onStale();
    },
  });
  const deny = useMutation({
    mutationFn: () => denyDeviceLogin(login.userCode),
    async onSuccess() {
      await onChanged("denied");
    },
    onError(error) {
      toast.error(getErrorMessage(error, "Could not deny this login."));
      onStale();
    },
  });
  const busy = approve.isLoading || deny.isLoading;
  const grantable = login.grantablePermissions ?? [];

  const authorize = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    if (access.kind === "none") {
      setMissing(true);
      permissionsRef.current?.scrollIntoView({
        block: "center",
        behavior: window.matchMedia("(prefers-reduced-motion: reduce)").matches ? "auto" : "smooth",
      });
      permissionsRef.current?.focus({ preventScroll: true });
      return;
    }
    approve.mutate({
      userCode: login.userCode,
      permissions: buildUrns(workspaceId, accessPolicies(access)),
    });
  };

  return (
    <form onSubmit={authorize} className="flex w-full flex-col gap-8">
      <BrandHeader
        tone="pending"
        title="Sign in to the Unkey CLI"
        description="Approve this request so the CLI can work in your workspace."
      />
      <OnboardingCard>
        <OnboardingCardContent className="mt-0 flex flex-col gap-6">
          <Step
            number={1}
            title="Confirm the code"
            description="Check this code matches the one in your terminal."
          >
            <CodeBoxes userCode={login.userCode} />
          </Step>
          <Step
            number={2}
            title="Choose access"
            description="Pick what the CLI can do in this workspace."
          >
            <div className="flex flex-col gap-4">
              <FormField label="Workspace">
                {({ id }) => <WorkspacePicker id={id} userCode={login.userCode} disabled={busy} />}
              </FormField>
              {grantable.length === 0 ? (
                <p className="text-sm text-gray-11">
                  Your role has no permissions it can give to the CLI.
                </p>
              ) : (
                <PermissionsField
                  access={access}
                  onChange={(next) => {
                    setAccess(next);
                    setMissing(false);
                  }}
                  workspaceId={workspaceId}
                  grantable={grantable}
                  missing={missing}
                  disabled={busy}
                  triggerRef={permissionsRef}
                />
              )}
            </div>
          </Step>
          <p className="flex gap-2 text-sm font-medium text-gray-11">
            <IconCircleInfoOutline18
              aria-hidden="true"
              className="mt-0.5 size-3.5 shrink-0 text-gray-9"
            />
            Only authorize if you started this login. Whoever started it gets access to this
            workspace.
          </p>
        </OnboardingCardContent>
        <OnboardingCardFooter className="flex items-center justify-end gap-2">
          <Button
            type="button"
            variant="outline"
            size="md"
            disabled={busy}
            loading={deny.isLoading}
            onClick={() => deny.mutate()}
          >
            Deny
          </Button>
          <Button
            type="submit"
            variant="primary"
            size="md"
            disabled={busy || grantable.length === 0}
            loading={approve.isLoading}
          >
            Authorize CLI
          </Button>
        </OnboardingCardFooter>
      </OnboardingCard>
    </form>
  );
}

function WorkspacePicker({
  id,
  userCode,
  disabled,
}: { id?: string; userCode: string; disabled: boolean }) {
  const { workspace, switchWorkspace } = useWorkspace();
  const available = trpc.workspace.listAvailable.useQuery();
  const orgs = available.data ?? [];
  if (orgs.length < 2 || !workspace?.orgId) {
    return workspace?.name ?? null;
  }
  return (
    <Combobox
      id={id}
      aria-label="Workspace"
      className="w-full"
      disabled={disabled}
      options={orgs.map((org) => ({
        value: org.orgId,
        label: org.name,
        searchValue: org.name,
      }))}
      value={workspace.orgId}
      onSelect={(organizationId) => {
        switchWorkspace(organizationId, routes.cli.device({ user_code: userCode }));
      }}
    />
  );
}
