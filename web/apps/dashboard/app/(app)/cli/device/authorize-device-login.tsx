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
import { useWorkspace } from "@/providers/workspace-provider";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Button, toast } from "@unkey/ui";
import { type ReactNode, useState } from "react";
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
      ) : query.isError ? (
        <p className="max-w-md text-balance text-center text-sm leading-5 text-gray-11">
          {query.error instanceof Error ? query.error.message : "This login could not be loaded."}
        </p>
      ) : query.data ? (
        <DeviceLoginView
          login={query.data}
          workspaceName={workspace?.name ?? ""}
          workspaceId={workspace?.id ?? ""}
          isAdmin={user?.role === "admin"}
          onChanged={refresh}
        />
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
        This login already issued a root key. You can close this page.
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
    <p className="max-w-md text-balance text-center text-sm leading-5 text-gray-11">
      Your device has been successfully authenticated please return to your terminal.
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
  const approve = useMutation({
    mutationFn: approveDeviceLogin,
    async onSuccess() {
      toast.success(
        "Your device has been successfully authenticated please return to your terminal.",
      );
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
    approve.mutate({
      userCode: login.userCode,
      name: values.name.trim(),
      permissions: [...buildUrns(workspaceId, values.policies)],
    });
  });
  const busy = approve.isLoading || deny.isLoading;

  if (step === "code") {
    return (
      <div className="flex w-full flex-col items-center gap-6 text-center">
        <StepIntro title="Authorize CLI" description="Confirm this code matches the terminal." />
        <LoginContext login={login} workspaceName={workspaceName} />
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
            disabled={busy}
            onClick={() => setStep("permissions")}
          >
            Confirm
          </Button>
        </div>
      </div>
    );
  }

  return (
    <FormProvider {...form}>
      <form onSubmit={submit} className="flex w-full flex-col items-center gap-6">
        <StepIntro
          title="Authorize CLI"
          description={
            <>
              The root key receives only the permissions you select. A workspace admin has to
              approve the login.
            </>
          }
        />
        <div ref={bodyRef} className="w-full text-left">
          <KeyFields>
            <PolicyList />
          </KeyFields>
        </div>
        {isAdmin ? null : (
          <p className="max-w-md text-balance text-center text-sm leading-5 text-gray-11">
            Your role cannot create root keys in this workspace.
          </p>
        )}
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
            disabled={busy || !isAdmin}
            loading={approve.isLoading}
          >
            Allow
          </Button>
        </div>
      </form>
    </FormProvider>
  );
}
