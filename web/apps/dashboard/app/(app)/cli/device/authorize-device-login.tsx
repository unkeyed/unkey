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
import { trpc } from "@/lib/trpc/client";
import { routes } from "@/lib/navigation/routes";
import { useWorkspace } from "@/providers/workspace-provider";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { IconTriangleWarningOutline18 } from "@unkey/icons";
import {
  AlertBanner,
  AlertBannerDescription,
  AlertBannerTitle,
  Button,
  TimestampInfo,
  toast,
} from "@unkey/ui";
import { type FormEvent, type ReactNode, useState } from "react";
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

  const refresh = async () => {
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
        <p className="max-w-md text-balance text-center text-sm leading-5 text-gray-11">
          {query.error instanceof Error ? query.error.message : "This login could not be loaded."}
        </p>
      ) : null}
    </CenteredFlow>
  );
}

function CenteredFlow({ children, wide = false }: { children: ReactNode; wide?: boolean }) {
  return (
    <div className="flex w-full flex-1 flex-col items-center justify-center p-12">
      <div
        className={
          wide
            ? "flex w-full max-w-3xl flex-col items-center gap-6"
            : "flex w-full max-w-md flex-col items-center gap-6 text-center"
        }
      >
        {children}
      </div>
    </div>
  );
}

function StepIntro({ title, description }: { title: string; description?: ReactNode }) {
  return (
    <div className="flex flex-col items-center gap-2 text-center">
      <h1 className="font-semibold text-lg text-gray-12">{title}</h1>
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
  onChanged: () => Promise<void>;
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
    <div className="flex w-full flex-col items-center gap-6 text-center">
      <StepIntro title="Authorize CLI" />
      <LoginContext login={login} workspaceName={workspaceName} />
      <ClosedStatus login={login} />
    </div>
  );
}

function LoginContext({ login, workspaceName }: { login: DeviceLogin; workspaceName: string }) {
  const characters = formatUserCode(login.userCode)
    .split(" ")
    .filter((char) => char.length > 0)
    .map((char, position) => ({ char, id: `${position}-${char}` }));

  return (
    <div className="flex w-full flex-col items-center gap-6 text-center">
      <div className="flex flex-wrap justify-center gap-2">
        {characters.map(({ char, id }) => (
          <span
            key={id}
            className="flex size-9 items-center justify-center rounded-md border bg-gray-2 font-mono text-sm text-gray-12"
          >
            {char}
          </span>
        ))}
      </div>
      <dl className="flex flex-col items-center gap-1 text-sm leading-5">
        <div className="flex gap-2">
          <dt className="text-gray-11">Device</dt>
          <dd className="text-gray-12">{login.deviceName || "Unknown device"}</dd>
        </div>
        <div className="flex gap-2">
          <dt className="text-gray-11">Requested from</dt>
          <dd className="font-mono text-gray-12">{login.requesterIp || "Unknown address"}</dd>
        </div>
        {login.requesterUserAgent ? (
          <div className="flex gap-2">
            <dt className="text-gray-11">Client</dt>
            <dd className="max-w-xs truncate text-gray-12">{login.requesterUserAgent}</dd>
          </div>
        ) : null}
        <div className="flex gap-2">
          <dt className="text-gray-11">Started</dt>
          <dd className="text-gray-12">
            <TimestampInfo value={login.createdAt} displayType="relative" />
          </dd>
        </div>
        <div className="flex gap-2">
          <dt className="text-gray-11">Workspace</dt>
          <dd className="text-gray-12">{workspaceName}</dd>
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

const authenticatedCopy =
  "Your device has been successfully authenticated. Please return to your terminal.";

function Approved() {
  return (
    <p className="max-w-md text-balance text-center text-sm leading-5 text-gray-11">
      {authenticatedCopy}
    </p>
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
  onChanged: () => Promise<void>;
}) {
  const [step, setStep] = useState<"code" | "permissions">("code");
  const grantable = login.grantablePermissions ?? [];
  const [selected, setSelected] = useState<string[]>(grantable);
  const approve = useMutation({
    mutationFn: approveDeviceLogin,
    async onSuccess() {
      toast.success(authenticatedCopy);
      await onChanged();
    },
    onError(error) {
      toast.error(error instanceof Error ? error.message : "Could not approve this login.");
    },
  });
  const deny = useMutation({
    mutationFn: () => denyDeviceLogin(login.userCode),
    async onSuccess() {
      toast.success("Login denied.");
      await onChanged();
    },
    onError(error) {
      toast.error(error instanceof Error ? error.message : "Could not deny this login.");
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
        <LoginContext login={login} workspaceName={workspaceName} />
        <WorkspacePicker userCode={login.userCode} />
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
            Confirm
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
        <fieldset className="flex max-h-80 w-full flex-col gap-2 overflow-auto rounded-md border p-3 text-left">
          <legend className="px-1 text-sm text-gray-11">Permissions for {workspaceName}</legend>
          {grantable.map((permission) => (
            <label key={permission} className="flex items-start gap-2 text-sm text-gray-12">
              <input
                type="checkbox"
                className="mt-1"
                checked={selected.includes(permission)}
                onChange={() =>
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
        </fieldset>
        {isAdmin ? (
          <div ref={bodyRef} className="w-full text-left">
            <p className="mb-3 text-sm text-gray-11">
              Add a narrower set below to replace full access. Leave it empty to keep the
              permissions selected above.
            </p>
            <KeyFields>
              <PolicyList />
            </KeyFields>
          </div>
        ) : null}
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
            type="submit"
            variant="primary"
            disabled={busy || nothingSelected}
            loading={approve.isLoading}
          >
            Allow
          </Button>
        </div>
        <LinkToSessions />
      </form>
    </FormProvider>
  );
}

function WorkspacePicker({ userCode }: { userCode: string }) {
  const { workspace } = useWorkspace();
  const available = trpc.workspace.listAvailable.useQuery();
  const orgs = available.data ?? [];
  if (orgs.length < 2 || !workspace?.orgId) {
    return null;
  }
  return (
    <label className="flex items-center gap-2 text-sm text-gray-12">
      Workspace
      <select
        className="rounded-md border bg-gray-2 px-2 py-1"
        value={workspace.orgId}
        onChange={(event) => {
          const organizationId = event.target.value;
          if (organizationId === workspace.orgId) {
            return;
          }
          const returnTo = routes.cli.device({ user_code: userCode });
          window.location.assign(
            routes.auth.switchOrganization({
              organizationId,
              returnTo,
            }),
          );
        }}
      >
        {orgs.map((org) => (
          <option key={org.orgId} value={org.orgId}>
            {org.name}
          </option>
        ))}
      </select>
    </label>
  );
}

function LinkToSessions() {
  return (
    <a href={routes.cli.sessions()} className="text-sm text-gray-11 underline">
      Manage CLI sessions
    </a>
  );
}
