"use client";

import { useRootKeyPolicyForm } from "@/app/(app)/[workspaceSlug]/settings/root-keys/components/builder/hooks/use-root-key-policy-form";
import { KeyFields } from "@/app/(app)/[workspaceSlug]/settings/root-keys/components/builder/key-fields";
import { buildUrns } from "@/app/(app)/[workspaceSlug]/settings/root-keys/components/builder/lib/urn";
import { PolicyList } from "@/app/(app)/[workspaceSlug]/settings/root-keys/components/builder/policy-list";
import type { RootKeyFormValues } from "@/app/(app)/[workspaceSlug]/settings/root-keys/components/builder/schema";
import { LoadingState } from "@/components/loading-state";
import {
  type DeviceLogin,
  approveDeviceLogin,
  denyDeviceLogin,
  getDeviceLogin,
} from "@/lib/cli/device-api";
import { formatUserCode } from "@/lib/cli/format-user-code";
import { routes } from "@/lib/navigation/routes";
import { trpc } from "@/lib/trpc/client";
import { getErrorMessage } from "@/lib/unkey-client";
import { useWorkspace } from "@/providers/workspace-provider";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { IconTriangleWarningOutline18 } from "@unkey/icons";
import {
  AlertBanner,
  AlertBannerDescription,
  AlertBannerTitle,
  Button,
  Checkbox,
  Combobox,
  TimestampInfo,
  toast,
} from "@unkey/ui";
import { type FormEvent, type ReactNode, useEffect, useRef, useState } from "react";
import { FormProvider } from "react-hook-form";

type AuthorizeDeviceLoginProps = {
  userCode: string;
};

export function AuthorizeDeviceLogin({ userCode }: AuthorizeDeviceLoginProps) {
  const { workspace, user } = useWorkspace();
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
    await queryClient.invalidateQueries({ queryKey: ["cli-device", trimmed] });
  };

  if (query.isLoading && trimmed.length > 0) {
    return <LoadingState message="Loading login..." />;
  }

  return (
    <CenteredFlow wide={query.data?.status === "pending"}>
      {trimmed.length === 0 ? (
        <StepIntro
          title="Authorize CLI"
          description={
            <>
              This page needs the code from <span className="font-mono">unkey login</span>. Run the
              command again and open the URL it prints.
            </>
          }
        />
      ) : query.data ? (
        <DeviceLoginView
          login={query.data}
          workspaceName={workspace?.name ?? ""}
          workspaceId={workspace?.id ?? ""}
          isAdmin={user?.role === "admin"}
          onChanged={refresh}
        />
      ) : query.isError ? (
        <div className="flex flex-col items-center gap-6">
          <StepIntro
            title="Could not load login"
            description={getErrorMessage(query.error, "This login could not be loaded.")}
          />
          <Button variant="outline" onClick={() => void query.refetch()}>
            Try again
          </Button>
        </div>
      ) : null}
    </CenteredFlow>
  );
}

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

function StepIntro({ title, description }: { title: string; description?: ReactNode }) {
  const headingRef = useRef<HTMLHeadingElement>(null);
  useEffect(() => {
    headingRef.current?.focus();
  }, []);

  return (
    <div className="flex flex-col items-center gap-2 text-center">
      <h1
        ref={headingRef}
        tabIndex={-1}
        className="font-semibold text-lg text-gray-12 outline-hidden"
      >
        {title}
      </h1>
      {description ? (
        <p className="max-w-md text-balance text-sm leading-5 text-gray-11">{description}</p>
      ) : null}
    </div>
  );
}

function DeviceLoginView({
  login,
  workspaceName,
  workspaceId,
  isAdmin,
  onChanged,
}: {
  login: DeviceLogin;
  workspaceName: string;
  workspaceId: string;
  isAdmin: boolean;
  onChanged: (status: "approved" | "denied") => Promise<void>;
}) {
  if (login.status === "approved") {
    return <Approved />;
  }
  if (login.status === "pending") {
    return (
      <PendingLogin
        login={login}
        workspaceName={workspaceName}
        workspaceId={workspaceId}
        isAdmin={isAdmin}
        onChanged={onChanged}
      />
    );
  }

  return (
    <div aria-live="polite" className="flex w-full flex-col items-center gap-6 text-center">
      <StepIntro title="Authorize CLI" />
      <LoginContext login={login} workspace={workspaceName} />
      <ClosedStatus login={login} />
    </div>
  );
}

function LoginContext({ login, workspace }: { login: DeviceLogin; workspace: ReactNode }) {
  const characters = Array.from(formatUserCode(login.userCode), (char, position) => ({
    char,
    id: `${position}-${char}`,
  }));

  return (
    <div className="flex w-full flex-col items-center gap-6 text-center">
      <div className="flex items-center justify-center gap-1 sm:gap-2">
        <span className="sr-only">Login code {formatUserCode(login.userCode)}</span>
        {characters.map(({ char, id }) => (
          <span
            key={id}
            aria-hidden="true"
            className={
              char === "-"
                ? "font-mono text-base text-gray-11"
                : "flex h-10 w-7 items-center justify-center rounded-md border border-grayA-4 bg-grayA-2 font-mono text-base text-gray-12 sm:w-10"
            }
          >
            {char}
          </span>
        ))}
      </div>
      <dl className="w-full divide-y divide-grayA-4 rounded-lg border border-grayA-4 px-4 text-left text-sm leading-5 [&>div]:grid [&>div]:grid-cols-[7rem_minmax(0,1fr)] [&>div]:gap-3 [&>div]:py-3 [&_dd]:min-w-0 [&_dd]:break-words">
        <div>
          <dt className="text-gray-11">Device</dt>
          <dd className="text-gray-12">{login.deviceName || "Unknown device"}</dd>
        </div>
        <div>
          <dt className="text-gray-11">Requested from</dt>
          <dd className="font-mono text-gray-12">{login.requesterIp || "Unknown address"}</dd>
        </div>
        {login.requesterUserAgent ? (
          <div>
            <dt className="text-gray-11">Client</dt>
            <dd className="text-gray-12">{login.requesterUserAgent}</dd>
          </div>
        ) : null}
        <div>
          <dt className="text-gray-11">Started</dt>
          <dd className="text-gray-12">
            <TimestampInfo value={login.createdAt} displayType="relative" />
          </dd>
        </div>
        <div className="items-center">
          <dt className="whitespace-nowrap text-gray-11">Workspace</dt>
          <dd className="text-gray-12">{workspace}</dd>
        </div>
      </dl>
    </div>
  );
}

function ClosedStatus({ login }: { login: DeviceLogin }) {
  if (login.status === "consumed") {
    return (
      <p className="max-w-md text-balance text-sm leading-5 text-gray-11">
        This login already issued a CLI session. You can close this page.
      </p>
    );
  }
  if (login.status === "denied") {
    return (
      <p className="max-w-md text-balance text-sm leading-5 text-gray-11">
        This login was denied. Run unkey login again to start over.
      </p>
    );
  }
  return (
    <p className="max-w-md text-balance text-sm leading-5 text-gray-11">
      This code expired. Run unkey login again.
    </p>
  );
}

function Approved() {
  return (
    <div aria-live="polite" className="flex flex-col items-center gap-6">
      <StepIntro
        title="CLI authorized"
        description="Return to your terminal. You can close this page."
      />
      <LinkToSessions />
    </div>
  );
}

function PendingLogin({
  login,
  workspaceName,
  workspaceId,
  isAdmin,
  onChanged,
}: {
  login: DeviceLogin;
  workspaceName: string;
  workspaceId: string;
  isAdmin: boolean;
  onChanged: (status: "approved" | "denied") => Promise<void>;
}) {
  const [step, setStep] = useState<"code" | "permissions">("code");
  const grantable = login.grantablePermissions ?? [];
  const [selected, setSelected] = useState<string[]>(grantable);
  const approve = useMutation({
    mutationFn: approveDeviceLogin,
    async onSuccess() {
      await onChanged("approved");
    },
    onError(error) {
      toast.error(getErrorMessage(error, "Could not approve this login."));
    },
  });
  const deny = useMutation({
    mutationFn: () => denyDeviceLogin(login.userCode),
    async onSuccess() {
      await onChanged("denied");
    },
    onError(error) {
      toast.error(getErrorMessage(error, "Could not deny this login."));
    },
  });

  const defaults: RootKeyFormValues = {
    name: login.deviceName ? `CLI (${login.deviceName})` : "CLI",
    policies: [],
  };
  const { form, bodyRef, submit } = useRootKeyPolicyForm(defaults, (values) => {
    const narrowed = [...buildUrns(workspaceId, values.policies)];
    approve.mutate({ userCode: login.userCode, permissions: narrowed });
  });
  const policies = form.watch("policies") ?? [];
  const allow = (event: FormEvent<HTMLFormElement>) => {
    if (policies.length > 0) {
      void submit(event);
      return;
    }
    event.preventDefault();
    if (selected.length === 0) {
      return;
    }
    approve.mutate({ userCode: login.userCode, permissions: selected });
  };
  const busy = approve.isLoading || deny.isLoading;
  const nothingSelected = selected.length === 0 && policies.length === 0;

  if (step === "code") {
    return (
      <div className="flex w-full flex-col items-center gap-6 text-center">
        <StepIntro title="Authorize CLI" description="Confirm this code matches the terminal." />
        <LoginContext
          login={login}
          workspace={<WorkspacePicker userCode={login.userCode} disabled={busy} />}
        />
        <AlertBanner variant="warning" className="text-left">
          <IconTriangleWarningOutline18 className="size-3.5" />
          <AlertBannerTitle>Only continue if you started this login</AlertBannerTitle>
          <AlertBannerDescription>
            Allowing it gives a CLI session to whoever ran unkey login. Deny it if someone sent you
            this link or the address above is not yours.
          </AlertBannerDescription>
        </AlertBanner>
        <div className="flex items-center justify-center gap-2">
          <Button
            type="button"
            variant="outline"
            disabled={busy}
            loading={deny.isLoading}
            onClick={() => deny.mutate()}
          >
            Deny
          </Button>
          <Button
            type="button"
            variant="primary"
            disabled={busy || grantable.length === 0}
            onClick={() => setStep("permissions")}
          >
            Review permissions
          </Button>
        </div>
        {grantable.length === 0 ? (
          <p className="max-w-md text-balance text-center text-sm leading-5 text-gray-11">
            Your role does not include any permissions you can grant to the CLI.
          </p>
        ) : null}
      </div>
    );
  }

  return (
    <FormProvider {...form}>
      <form onSubmit={allow} className="flex w-full flex-col items-center gap-6">
        <StepIntro
          title="Authorize CLI"
          description="Everything your role allows is selected. Uncheck a permission to narrow the session."
        />
        <fieldset
          disabled={busy}
          className="w-full min-w-0 rounded-lg border border-grayA-4 text-left disabled:opacity-50"
        >
          <legend className="mx-3 px-1 text-sm font-medium text-gray-12">
            Permissions for {workspaceName}
          </legend>
          <div className="max-h-80 overflow-auto p-2">
            {grantable.map((permission) => (
              <label
                key={permission}
                htmlFor={`cli-permission-${permission}`}
                className="flex cursor-pointer items-start gap-3 rounded-md p-2 text-sm text-gray-12 hover:bg-grayA-2"
              >
                <Checkbox
                  id={`cli-permission-${permission}`}
                  className="mt-0.5"
                  disabled={busy || policies.length > 0}
                  checked={selected.includes(permission)}
                  onCheckedChange={() =>
                    setSelected((current) =>
                      current.includes(permission)
                        ? current.filter((item) => item !== permission)
                        : [...current, permission],
                    )
                  }
                />
                <span className="break-all font-mono text-xs">{permission}</span>
              </label>
            ))}
          </div>
        </fieldset>
        {isAdmin ? (
          <div ref={bodyRef} className="w-full text-left">
            <fieldset disabled={busy} className="min-w-0 disabled:opacity-50">
              <p className="mb-3 text-sm text-gray-11">
                Add a narrower set below to replace full access. Leave it empty to keep the
                permissions selected above.
              </p>
              <KeyFields nameHidden>
                <PolicyList />
              </KeyFields>
            </fieldset>
          </div>
        ) : null}
        <div className="flex items-center justify-center gap-2">
          <Button type="button" variant="ghost" disabled={busy} onClick={() => setStep("code")}>
            Back
          </Button>
          <Button
            type="button"
            variant="outline"
            disabled={busy}
            loading={deny.isLoading}
            onClick={() => deny.mutate()}
          >
            Deny
          </Button>
          <Button
            type="submit"
            variant="primary"
            disabled={busy || nothingSelected}
            loading={approve.isLoading}
          >
            Authorize CLI
          </Button>
        </div>
        <LinkToSessions />
      </form>
    </FormProvider>
  );
}

function WorkspacePicker({ userCode, disabled }: { userCode: string; disabled: boolean }) {
  const { workspace, switchWorkspace } = useWorkspace();
  const available = trpc.workspace.listAvailable.useQuery();
  const orgs = available.data ?? [];
  if (orgs.length < 2 || !workspace?.orgId) {
    return workspace?.name ?? null;
  }
  return (
    <Combobox
      aria-label="Workspace"
      className="w-full max-w-60"
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

function LinkToSessions() {
  return (
    <a href={routes.cli.sessions()} className="text-sm text-gray-11 underline">
      Manage CLI sessions
    </a>
  );
}
