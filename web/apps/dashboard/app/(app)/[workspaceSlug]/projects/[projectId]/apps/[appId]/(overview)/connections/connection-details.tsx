"use client";

import { trpc } from "@/lib/trpc/client";
import {
  connectionHost,
  connectionHostVariable,
  connectionNameSchema,
} from "@/lib/trpc/routers/deploy/app-connection/validation";
import {
  IconChevronRightOutline18,
  IconCubeOutline18,
  IconPenWriting3Outline18,
  IconTrashOutline18,
  IconXmarkOutline18,
} from "@unkey/icons";
import { Button, CopyButton, cn, toast } from "@unkey/ui";
import { useLayoutEffect, useRef, useState } from "react";
import {
  type ConnectionTargets,
  type Environment,
  type ListedConnection,
  type TargetRule,
  describeTarget,
  isProduction,
  ruleOf,
} from "./connection-rules";

function CopyValue({ label, value }: { label: string; value: string }) {
  return (
    <fieldset className="min-w-0 space-y-1.5">
      <legend className="break-all text-xs text-gray-11">{label}</legend>
      <div className="flex min-w-0 items-start gap-2 rounded-lg border border-gray-4 bg-gray-2 px-2 py-1.5">
        <code className="min-w-0 flex-1 whitespace-pre-wrap break-all py-1 text-xs leading-5">
          {value}
        </code>
        <CopyButton value={value} variant="ghost" className="shrink-0" />
      </div>
    </fieldset>
  );
}

export function ConnectionDetails({
  projectId,
  appName,
  connection,
  environment,
  targets,
  hasMoreDeployments,
  loadingMoreDeployments,
  onLoadMoreDeployments,
  onClose,
  onRemove,
  onSaved,
}: {
  projectId: string;
  appName: string;
  connection: ListedConnection;
  environment: Environment;
  targets: ConnectionTargets;
  hasMoreDeployments: boolean;
  loadingMoreDeployments: boolean;
  onLoadMoreDeployments: () => void;
  onClose: () => void;
  onRemove: () => void;
  onSaved: () => Promise<void>;
}) {
  const [pickingDeployment, setPickingDeployment] = useState(
    connection.targetType === "deployment",
  );
  const [renaming, setRenaming] = useState(false);
  const [draftName, setDraftName] = useState(connection.name);
  const [language, setLanguage] = useState("Node.js");
  const renameInput = useRef<HTMLInputElement>(null);
  useLayoutEffect(() => {
    if (renaming) {
      renameInput.current?.focus();
    }
  }, [renaming]);
  const update = trpc.appConnection.update.useMutation({
    onSuccess: async (result) => {
      if (result.name !== connection.name) {
        toast.success("Connection renamed", {
          description: `Update your code and redeploy ${appName} to use ${connectionHostVariable(result.name)}.`,
        });
      }
      setRenaming(false);
      await onSaved();
    },
    onError: (error) => toast.error(error.message),
  });
  const targetName = connection.targetAppName;
  const rule = ruleOf(connection);
  const automaticProduction = isProduction(environment);
  const targetEnvironments = targets.environments.filter(
    (env) => env.appId === connection.targetAppId,
  );
  const productionEnvironment = targetEnvironments.find(isProduction);
  const targetDeployments = targets.deployments.filter(
    (item) => item.appId === connection.targetAppId,
  );
  const description = describeTarget({
    connection,
    targetName,
    callerEnvironment: environment,
    environments: targets.environments,
    deployments: targets.deployments,
  });
  const save = (next: TargetRule, name?: string) =>
    update.mutate({
      projectId,
      id: connection.id,
      ...next,
      ...(name ? { name } : {}),
    });
  const selectValue =
    pickingDeployment || connection.targetType === "deployment"
      ? "deployment"
      : connection.targetType === "environment"
        ? automaticProduction && connection.targetEnvironmentId === productionEnvironment?.id
          ? "automatic"
          : `env:${connection.targetEnvironmentId}`
        : "automatic";
  const nameCheck = connectionNameSchema.safeParse(draftName);
  const variable = connectionHostVariable(connection.name);
  const snippet =
    language === "Node.js"
      ? `const host =\n  process.env.${variable};`
      : language === "Go"
        ? `host := os.Getenv("${variable}")`
        : `host = os.environ["${variable}"]`;

  return (
    <div className="divide-y divide-gray-4">
      <div className="flex items-center gap-2 p-4">
        <IconCubeOutline18 className="size-4 shrink-0" />
        <div className="min-w-0 flex-1">
          <h2 className="break-words text-sm font-medium">{targetName}</h2>
          <p className="mt-1 break-words text-xs text-gray-9">
            {appName} → {targetName}
          </p>
        </div>
        <Button variant="ghost" size="icon" aria-label="Close connection details" onClick={onClose}>
          <IconXmarkOutline18 className="size-4" />
        </Button>
      </div>
      <div className="space-y-3 p-4">
        <h3 className="text-xs font-medium">Use in {appName}</h3>
        <dl className="divide-y divide-gray-4 rounded-lg border border-gray-4">
          {[
            { label: "Variable", value: variable },
            { label: "Hostname", value: connectionHost(connection.name) },
          ].map(({ label, value }) => (
            <div
              key={label}
              className="grid grid-cols-[64px_minmax(0,1fr)] items-start gap-2 px-2 py-2"
            >
              <dt className="whitespace-nowrap py-1 text-xs leading-5 text-gray-11">{label}</dt>
              <dd className="flex min-w-0 items-start gap-2">
                <code className="min-w-0 flex-1 break-all py-1 text-xs leading-5">{value}</code>
                <CopyButton value={value} variant="ghost" className="shrink-0" />
              </dd>
            </div>
          ))}
        </dl>
        <p className="text-xs leading-5 text-gray-9">
          Unkey sets this variable. Hostname only, not a URL.
        </p>
        <p className="text-xs leading-5 text-gray-9">
          New or renamed variables arrive on {appName}’s next deployment.
        </p>
      </div>
      <div className="space-y-3 p-4">
        <label className="flex items-center justify-between gap-3 text-xs font-medium">
          Target
          <select
            aria-label={`Target for ${connection.name}`}
            className="h-8 min-w-0 max-w-[75%] rounded-md border border-gray-5 bg-background px-2 text-xs font-normal text-gray-11"
            value={selectValue}
            disabled={update.isLoading}
            onChange={(event) => {
              const value = event.target.value;
              if (value === "deployment") {
                setPickingDeployment(true);
                return;
              }
              setPickingDeployment(false);
              save(
                value === "automatic"
                  ? { targetType: "automatic" }
                  : {
                      targetType: "environment",
                      targetEnvironmentId: value.slice("env:".length),
                    },
              );
            }}
          >
            <option value="automatic">
              {automaticProduction ? "production" : "Same Git branch"}
            </option>
            {connection.targetType === "environment" &&
              !targetEnvironments.some((env) => env.id === connection.targetEnvironmentId) && (
                <option value={`env:${connection.targetEnvironmentId}`} disabled>
                  Deleted environment
                </option>
              )}
            {targetEnvironments
              .filter((env) => !automaticProduction || !isProduction(env))
              .map((env) => (
                <option key={env.id} value={`env:${env.id}`}>
                  {env.slug}
                </option>
              ))}
            {(targetDeployments.length > 0 || connection.targetType === "deployment") && (
              <option value="deployment">Specific deployment…</option>
            )}
          </select>
        </label>
        {pickingDeployment && (
          <select
            aria-label={`Deployment for ${connection.name}`}
            className="h-7 w-full rounded-md border border-gray-5 bg-background px-2 text-xs"
            value={
              connection.targetType === "deployment" ? (connection.targetDeploymentId ?? "") : ""
            }
            disabled={update.isLoading}
            onChange={(event) => {
              if (event.target.value) {
                save({
                  targetType: "deployment",
                  targetDeploymentId: event.target.value,
                });
              }
            }}
          >
            <option value="">Choose a deployment</option>
            {connection.targetDeploymentId &&
              !targetDeployments.some((item) => item.id === connection.targetDeploymentId) && (
                <option value={connection.targetDeploymentId} disabled>
                  Unavailable deployment
                </option>
              )}
            {targetDeployments.map((item) => (
              <option key={item.id} value={item.id} disabled={item.status !== "ready"}>
                {`${item.id} · ${item.gitBranch ?? item.image ?? item.status}${item.status === "ready" ? "" : " (stopped)"}`}
              </option>
            ))}
          </select>
        )}
        {pickingDeployment && hasMoreDeployments && (
          <Button
            variant="outline"
            size="sm"
            disabled={loadingMoreDeployments}
            onClick={onLoadMoreDeployments}
          >
            {loadingMoreDeployments ? "Loading deployments…" : "Load more deployments"}
          </Button>
        )}
        <p
          className={cn(
            "break-words text-xs leading-5 text-gray-9",
            description.unavailable && "rounded-lg bg-warning-3 p-3 text-warning-11",
          )}
        >
          {description.text}
        </p>
        <p className="text-xs leading-5 text-gray-9">
          Target changes apply to running deployments.
        </p>
      </div>
      <details className="group/code">
        <summary className="flex cursor-pointer list-none items-center gap-2 p-4 text-xs font-medium focus-visible:outline-2 focus-visible:outline-gray-8 [&::-webkit-details-marker]:hidden">
          <IconChevronRightOutline18 className="size-4 shrink-0 group-open/code:rotate-90" />
          Read in code
          <span className="ml-auto text-xs font-normal text-gray-9">Node.js, Go, Python</span>
        </summary>
        <div className="space-y-3 px-4 pb-4">
          <label className="flex items-center justify-between gap-2 text-xs text-gray-11">
            Language
            <select
              aria-label="Code language"
              value={language}
              onChange={(event) => setLanguage(event.target.value)}
              className="h-7 rounded-md border border-gray-5 bg-background px-2 text-xs"
            >
              <option>Node.js</option>
              <option>Go</option>
              <option>Python</option>
            </select>
          </label>
          <CopyValue
            label={
              language === "Node.js"
                ? "Read the environment variable"
                : language === "Go"
                  ? 'Using the Go "os" package'
                  : "Using the Python os module"
            }
            value={snippet}
          />
          <p className="text-xs leading-5 text-gray-9">
            Pass the hostname to your client with {targetName}’s protocol and port.
          </p>
        </div>
      </details>
      <div className="space-y-3 p-4 text-xs">
        <div className="flex flex-wrap items-center gap-2">
          {renaming ? (
            <form
              className="flex min-w-0 flex-1 items-center gap-1.5"
              onSubmit={(event) => {
                event.preventDefault();
                if (rule && nameCheck.success) {
                  save(rule, nameCheck.data);
                }
              }}
            >
              <input
                ref={renameInput}
                aria-label={`Name for ${connection.name}`}
                className="h-7 min-w-0 flex-1 rounded-md border border-gray-5 bg-background px-2 font-mono text-xs"
                value={draftName}
                onChange={(event) => setDraftName(event.target.value)}
                onKeyDown={(event) => {
                  if (event.key === "Escape") {
                    setRenaming(false);
                    setDraftName(connection.name);
                  }
                }}
              />
              <Button
                type="submit"
                variant="outline"
                size="sm"
                disabled={!rule || !nameCheck.success || update.isLoading}
              >
                Save
              </Button>
              <Button
                type="button"
                variant="ghost"
                size="sm"
                disabled={update.isLoading}
                onClick={() => setRenaming(false)}
              >
                Cancel
              </Button>
            </form>
          ) : (
            <>
              <Button
                variant="outline"
                size="sm"
                disabled={update.isLoading}
                onClick={() => {
                  setDraftName(connection.name);
                  setRenaming(true);
                }}
              >
                <IconPenWriting3Outline18 className="size-4" />
                Rename
              </Button>
              <Button
                variant="ghost"
                size="sm"
                aria-label={`Remove ${connection.name} connection`}
                disabled={update.isLoading}
                onClick={onRemove}
              >
                <IconTrashOutline18 className="size-4" />
                Remove
              </Button>
            </>
          )}
        </div>
        {renaming && (
          <p className="leading-5 text-gray-9">
            Renaming changes the hostname and variable name. Update your code before redeploying.
          </p>
        )}
        {renaming && !nameCheck.success && (
          <p className="text-warning-11">{nameCheck.error.issues[0]?.message}</p>
        )}
      </div>
    </div>
  );
}
