"use client";

import { collection } from "@/lib/collections";
import { trpc } from "@/lib/trpc/client";
import {
  bindingHost,
  bindingHostVariable,
  bindingNameSchema,
} from "@/lib/trpc/routers/deploy/app-binding/validation";
import { and, eq, useLiveQuery } from "@tanstack/react-db";
import {
  IconCheckOutline18,
  IconChevronRightOutline18,
  IconCubeOutline18,
  IconLinkOutline18,
  IconMagnifierOutline18,
  IconPenWriting3Outline18,
  IconPlusOutline18,
  IconTrashOutline18,
  IconXmarkOutline18,
} from "@unkey/icons";
import {
  Button,
  CopyButton,
  DialogContainer,
  InputGroup,
  InputGroupAddon,
  InputGroupInput,
  PageBody,
  PageContainer,
  PageHeader,
  PageHeaderActions,
  PageHeaderContent,
  PageHeaderDescription,
  PageHeaderTitle,
  Popover,
  PopoverContent,
  PopoverTrigger,
  Skeleton,
  cn,
  toast,
} from "@unkey/ui";
import { type ReactNode, useId, useLayoutEffect, useRef, useState } from "react";
import { useAppId, useProjectData } from "../data-provider";
import {
  type Binding,
  type Deployment,
  type Environment,
  type TargetRule,
  describeTarget,
  isProduction,
  ruleOf,
} from "./binding-rules";

type Targets = {
  apps: { id: string; name: string; slug: string }[];
  environments: Environment[];
  deployments: (Deployment & { gitBranch: string | null; image: string | null })[];
};
type ListedBinding = Binding & { name: string; targetAppName: string };
type Wire = { id: string; d: string };

export default function BindingsPage() {
  const { projectId, environments, isEnvironmentsLoading } = useProjectData();
  const appId = useAppId();
  const [selectedEnvironment, setSelectedEnvironment] = useState<string | null>(null);
  const environment =
    environments.find((env) => env.id === selectedEnvironment) ??
    environments.find((env) => env.slug === "production") ??
    environments[0];

  return (
    <PageContainer>
      {isEnvironmentsLoading || !environment ? (
        <>
          <BindingsHeader />
          <PageBody>
            {isEnvironmentsLoading ? (
              <Skeleton className="h-96 w-full" />
            ) : (
              <p className="text-sm text-gray-9">Create an environment before adding bindings.</p>
            )}
          </PageBody>
        </>
      ) : (
        <BindingCanvas
          key={environment.id}
          projectId={projectId}
          appId={appId}
          environment={environment}
          environments={environments}
          onEnvironmentChange={setSelectedEnvironment}
        />
      )}
    </PageContainer>
  );
}

function BindingsHeader({ children }: { children?: ReactNode }) {
  return (
    <PageHeader>
      <PageHeaderContent>
        <PageHeaderTitle>Bindings</PageHeaderTitle>
        <PageHeaderDescription>Connect this app to the services it needs.</PageHeaderDescription>
      </PageHeaderContent>
      {children && <PageHeaderActions>{children}</PageHeaderActions>}
    </PageHeader>
  );
}

function BindingCanvas({
  projectId,
  appId,
  environment,
  environments,
  onEnvironmentChange,
}: {
  projectId: string;
  appId: string;
  environment: Environment;
  environments: Environment[];
  onEnvironmentChange: (id: string) => void;
}) {
  const utils = trpc.useUtils();
  const appQuery = useLiveQuery(
    (q) =>
      q
        .from({ app: collection.apps })
        .where(({ app }) => and(eq(app.projectId, projectId), eq(app.id, appId))),
    [projectId, appId],
  );
  const bindings = trpc.appBinding.list.useQuery({
    projectId,
    appId,
    environmentId: environment.id,
  });
  const targets = trpc.appBinding.targets.useQuery({ projectId, appId });
  const [deletingId, setDeletingId] = useState<string | null>(null);
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const [search, setSearch] = useState("");
  const [canvasRoot, setCanvasRoot] = useState<HTMLDivElement | null>(null);
  const [pickerOpen, setPickerOpen] = useState(false);
  const refresh = () => utils.appBinding.list.invalidate({ projectId });
  const create = trpc.appBinding.create.useMutation({
    onSuccess: async (result) => {
      setPickerOpen(false);
      setSearch("");
      await refresh();
      setSelectedId(result.id);
      toast.success("Binding created", {
        description: `Redeploy ${appQuery.data?.[0]?.name ?? "this app"} to add ${bindingHostVariable(result.name)}.`,
      });
    },
    onError: (error) => toast.error(error.message),
  });
  const remove = trpc.appBinding.delete.useMutation({
    onSuccess: async () => {
      setDeletingId(null);
      setSelectedId(null);
      await refresh();
      toast.success("Binding deleted");
    },
    onError: (error) => toast.error(error.message),
  });
  if (bindings.isLoading || targets.isLoading || appQuery.isLoading) {
    return (
      <>
        <BindingsHeader />
        <PageBody>
          <Skeleton aria-label="Loading bindings" className="h-96 w-full" />
        </PageBody>
      </>
    );
  }
  if (bindings.isError || targets.isError || appQuery.isError || !appQuery.data[0]) {
    return (
      <>
        <BindingsHeader />
        <PageBody>
          <div role="alert" className="rounded-xl border border-gray-4 p-8 text-sm text-error-11">
            We couldn't load app bindings. Reload this page to try again.
          </div>
        </PageBody>
      </>
    );
  }

  const app = appQuery.data[0];
  const connected = new Set(bindings.data.map((binding) => binding.targetAppId));
  const others = targets.data.apps.filter((target) => !connected.has(target.id));
  const deleting = bindings.data.find((binding) => binding.id === deletingId);
  const selected = bindings.data.find((binding) => binding.id === selectedId);
  const query = search.trim().toLowerCase();
  const visible = bindings.data.filter((binding) =>
    `${binding.targetAppName} ${binding.name} ${bindingHostVariable(binding.name)}`
      .toLowerCase()
      .includes(query),
  );

  return (
    <>
      <BindingsHeader>
        <label className="flex items-center gap-2 text-xs text-gray-9">
          Environment
          <select
            aria-label="Caller environment"
            className="h-9 max-w-[180px] rounded-lg border border-gray-5 bg-background px-3 text-sm text-gray-11"
            value={environment.id}
            onChange={(event) => onEnvironmentChange(event.target.value)}
          >
            {environments.map((env) => (
              <option key={env.id} value={env.id}>
                {env.slug}
              </option>
            ))}
          </select>
        </label>
      </BindingsHeader>
      <PageBody>
        <div
          className={cn(
            "grid min-w-0 items-start gap-5",
            selected && "lg:grid-cols-[minmax(0,1fr)_320px]",
          )}
        >
          <div
            className="@container min-w-0 rounded-xl border border-gray-4 bg-gray-3/40"
            aria-label="Binding canvas"
          >
            <div
              ref={setCanvasRoot}
              className="relative flex min-h-52 flex-col items-start gap-8 p-5 @min-[460px]:flex-row @min-[460px]:gap-10"
            >
              <button
                type="button"
                aria-label="Clear binding selection"
                className="absolute inset-0 rounded-xl"
                tabIndex={-1}
                disabled={!selected}
                onClick={() => setSelectedId(null)}
              />
              <Group
                icon={<IconCubeOutline18 className="size-4" />}
                label="This app"
                className="w-full @min-[460px]:w-40"
              >
                <div data-wire-from className="rounded-lg border bg-raised px-3 py-2 shadow-xs">
                  <span className="block truncate text-sm font-medium" title={app.name}>
                    {app.name}
                  </span>
                </div>
              </Group>
              <Group
                icon={<IconLinkOutline18 className="size-4" />}
                label={
                  bindings.data.length ? `Connects to · ${bindings.data.length}` : "Connects to"
                }
                className="w-full pl-6 @min-[460px]:max-w-sm @min-[460px]:flex-1 @min-[460px]:pl-0"
              >
                {(bindings.data.length > 8 || search.length > 0) && (
                  <InputGroup className="h-8">
                    <InputGroupAddon>
                      <IconMagnifierOutline18 className="size-4" />
                    </InputGroupAddon>
                    <InputGroupInput
                      aria-label="Filter bindings"
                      placeholder="Filter bindings…"
                      value={search}
                      onChange={(event) => setSearch(event.target.value)}
                    />
                  </InputGroup>
                )}
                {visible.map((binding) => (
                  <BindingNode
                    key={binding.id}
                    binding={binding}
                    environment={environment}
                    targets={targets.data}
                    selected={selected?.id === binding.id}
                    onSelect={() => setSelectedId(selected?.id === binding.id ? null : binding.id)}
                  />
                ))}
                {bindings.data.length === 0 && (
                  <p className="py-2 text-sm text-gray-9">
                    No bindings yet. Add an app that {app.name} needs to call.
                  </p>
                )}
                {bindings.data.length > 0 && visible.length === 0 && (
                  <p className="py-2 text-sm text-gray-9">No matching bindings.</p>
                )}
                <AddBindingPicker
                  open={pickerOpen}
                  onOpenChange={setPickerOpen}
                  apps={others}
                  environments={targets.data.environments}
                  requiresEnvironment={app.sourceType === "oci" && !isProduction(environment)}
                  pending={create.isLoading}
                  onAdd={(targetAppId, rule) =>
                    create.mutate({
                      projectId,
                      appId,
                      environmentId: environment.id,
                      targetAppId,
                      ...rule,
                    })
                  }
                />
              </Group>
              <BindingWires
                key={visible.map((binding) => binding.id).join(",")}
                root={canvasRoot}
                highlightedId={selected?.id ?? null}
              />
            </div>
          </div>
          {selected && (
            <aside
              id="binding-details"
              aria-label="Binding details"
              className="min-w-0 break-words rounded-xl border border-gray-4 bg-background"
            >
              <BindingDetails
                key={selected.id}
                projectId={projectId}
                appName={app.name}
                binding={selected}
                environment={environment}
                targets={targets.data}
                onClose={() => {
                  canvasRoot?.querySelector<HTMLElement>('[aria-pressed="true"]')?.focus();
                  setSelectedId(null);
                }}
                onDelete={() => setDeletingId(selected.id)}
                onSaved={refresh}
              />
            </aside>
          )}
        </div>
      </PageBody>
      <DialogContainer
        isOpen={!!deleting}
        onOpenChange={(open) => {
          if (!open) {
            setDeletingId(null);
          }
        }}
        title={`Delete the ${deleting?.targetAppName ?? ""} binding?`}
        subTitle={
          deleting
            ? `${app.name} can no longer connect to ${deleting.targetAppName} once the change applies. ${bindingHostVariable(deleting.name)} stays in running deployments until you redeploy ${app.name}.`
            : undefined
        }
        footer={
          <div className="flex gap-2">
            <Button variant="outline" onClick={() => setDeletingId(null)}>
              Keep binding
            </Button>
            <Button
              variant="primary"
              color="danger"
              disabled={remove.isLoading}
              onClick={() => {
                if (deleting) {
                  remove.mutate({ projectId, id: deleting.id });
                }
              }}
            >
              Delete binding
            </Button>
          </div>
        }
      />
    </>
  );
}

function AddBindingPicker({
  open,
  onOpenChange,
  apps,
  environments,
  requiresEnvironment,
  pending,
  onAdd,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  apps: Targets["apps"];
  environments: Environment[];
  requiresEnvironment: boolean;
  pending: boolean;
  onAdd: (appId: string, rule: TargetRule) => void;
}) {
  const [step, setStep] = useState<"resource" | "app">("resource");
  const [search, setSearch] = useState("");
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const [environmentId, setEnvironmentId] = useState("");
  const searchInput = useRef<HTMLInputElement>(null);
  const selected = apps.find((app) => app.id === selectedId);
  const query = search.trim().toLowerCase();
  const visible = apps.filter((app) => `${app.name} ${app.slug}`.toLowerCase().includes(query));
  const targetEnvironments = environments.filter((env) => env.appId === selected?.id);
  const canAdd =
    selected !== undefined &&
    (!requiresEnvironment || targetEnvironments.some((env) => env.id === environmentId));

  useLayoutEffect(() => {
    if (open && step === "app") {
      searchInput.current?.focus();
    }
  }, [open, step]);

  return (
    <Popover
      open={open}
      onOpenChange={(next) => {
        onOpenChange(next);
        if (next) {
          setStep("resource");
          setSearch("");
          setSelectedId(null);
          setEnvironmentId("");
        }
      }}
    >
      <PopoverTrigger
        disabled={pending}
        className="flex h-9 w-full items-center gap-2 rounded-lg border border-dashed border-gray-6 px-3 text-sm text-gray-11 hover:border-gray-8 hover:bg-gray-3 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-gray-8 disabled:opacity-50"
      >
        <IconPlusOutline18 className="size-4" />
        Add binding
      </PopoverTrigger>
      <PopoverContent
        side="bottom"
        align="start"
        sideOffset={12}
        className="w-72 max-w-[calc(100vw-2rem)] p-2"
        aria-label={step === "resource" ? "Add binding" : "Choose an app"}
      >
        {step === "resource" ? (
          <button
            type="button"
            className="flex w-full items-center gap-3 rounded-md p-2 text-left hover:bg-gray-3 focus-visible:outline focus-visible:outline-2 focus-visible:outline-gray-8"
            onClick={() => setStep("app")}
          >
            <IconCubeOutline18 className="size-4 text-gray-11" />
            <span>
              <span className="block text-sm font-medium text-gray-12">App</span>
              <span className="block text-xs text-gray-9">Private connection</span>
            </span>
          </button>
        ) : (
          <form
            onSubmit={(event) => {
              event.preventDefault();
              if (selected && canAdd && !pending) {
                onAdd(
                  selected.id,
                  requiresEnvironment
                    ? { targetType: "environment", targetEnvironmentId: environmentId }
                    : { targetType: "automatic" },
                );
              }
            }}
          >
            <h2 className="px-1 pb-3 pt-1 text-sm font-medium">Choose an app</h2>
            <InputGroup className="h-8">
              <InputGroupAddon>
                <IconMagnifierOutline18 className="size-4 text-gray-9" />
              </InputGroupAddon>
              <InputGroupInput
                ref={searchInput}
                aria-label="Search apps"
                placeholder="Search apps…"
                value={search}
                disabled={pending}
                onChange={(event) => {
                  setSearch(event.target.value);
                  setSelectedId(null);
                  setEnvironmentId("");
                }}
              />
            </InputGroup>
            <div
              className="my-2 max-h-48 overflow-y-auto"
              role="radiogroup"
              aria-label="Available apps"
            >
              {visible.map((app) => (
                <label
                  key={app.id}
                  className={cn(
                    "flex cursor-pointer items-center gap-2 rounded-md px-2 py-2 text-sm hover:bg-gray-3 has-focus-visible:outline has-focus-visible:outline-2 has-focus-visible:outline-gray-8",
                    selectedId === app.id && "bg-gray-3",
                  )}
                >
                  <input
                    type="radio"
                    name="binding-app"
                    value={app.id}
                    checked={selectedId === app.id}
                    disabled={pending}
                    className="sr-only"
                    onChange={() => {
                      setSelectedId(app.id);
                      setEnvironmentId("");
                    }}
                  />
                  <IconCubeOutline18 className="size-4 shrink-0 text-gray-9" />
                  <span className="min-w-0 flex-1 truncate">{app.name}</span>
                  {selectedId === app.id && <IconCheckOutline18 className="size-4 shrink-0" />}
                </label>
              ))}
              {visible.length === 0 && (
                <p className="px-2 py-3 text-xs text-gray-9">
                  {apps.length === 0 ? "All apps are already connected." : "No matching apps."}
                </p>
              )}
            </div>
            {selected && requiresEnvironment && (
              <label className="mb-3 flex flex-col gap-1.5 px-1 text-xs text-gray-11">
                Target environment
                <select
                  className="h-8 rounded-md border border-gray-5 bg-background px-2"
                  value={environmentId}
                  disabled={pending}
                  onChange={(event) => setEnvironmentId(event.target.value)}
                >
                  <option value="">Choose an environment</option>
                  {targetEnvironments.map((env) => (
                    <option key={env.id} value={env.id}>
                      {env.slug}
                    </option>
                  ))}
                </select>
              </label>
            )}
            <Button
              type="submit"
              className="w-full"
              disabled={!canAdd || pending}
              loading={pending}
            >
              Add binding
            </Button>
          </form>
        )}
      </PopoverContent>
    </Popover>
  );
}

function Group({
  icon,
  label,
  className,
  children,
}: {
  icon: ReactNode;
  label: string;
  className?: string;
  children: ReactNode;
}) {
  return (
    <div className={cn("relative flex min-w-0 flex-col gap-2", className)}>
      <span className="flex items-center gap-1.5 px-1 pb-0.5 text-xs text-gray-11">
        {icon}
        {label}
      </span>
      {children}
    </div>
  );
}

function BindingNode({
  binding,
  environment,
  targets,
  selected,
  onSelect,
}: {
  binding: ListedBinding;
  environment: Environment;
  targets: Targets;
  selected: boolean;
  onSelect: () => void;
}) {
  const description = describeTarget({
    binding,
    targetName: binding.targetAppName,
    callerEnvironment: environment,
    ...targets,
  });
  const targetLabel =
    binding.targetType === "automatic"
      ? isProduction(environment)
        ? "production"
        : "Same branch"
      : binding.targetType === "environment"
        ? (targets.environments.find((env) => env.id === binding.targetEnvironmentId)?.slug ??
          "Deleted environment")
        : binding.targetDeploymentId;
  return (
    <button
      type="button"
      data-wire-to={binding.id}
      data-binding={binding.id}
      aria-pressed={selected}
      aria-controls={selected ? "binding-details" : undefined}
      onClick={onSelect}
      className={cn(
        "flex w-full min-w-0 items-center gap-2 rounded-lg border bg-raised px-3 py-2 text-left shadow-xs transition-[border-color,box-shadow] hover:border-gray-8 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-gray-10",
        selected && "border-gray-10 shadow-[0_0_0_3px_var(--color-grayA-3)]",
      )}
    >
      <IconCubeOutline18 className="size-4 shrink-0 text-gray-11" />
      <span className="min-w-0 flex-1 truncate text-sm font-medium" title={binding.targetAppName}>
        {binding.targetAppName}
      </span>
      <span
        className={cn(
          "max-w-[45%] truncate rounded bg-gray-3 px-1.5 py-0.5 text-xs text-gray-11",
          description.unavailable && "bg-warning-3 text-warning-11",
        )}
        title={description.text}
      >
        {targetLabel}
      </span>
    </button>
  );
}

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

function BindingDetails({
  projectId,
  appName,
  binding,
  environment,
  targets,
  onClose,
  onDelete,
  onSaved,
}: {
  projectId: string;
  appName: string;
  binding: ListedBinding;
  environment: Environment;
  targets: Targets;
  onClose: () => void;
  onDelete: () => void;
  onSaved: () => Promise<void>;
}) {
  const [pickingDeployment, setPickingDeployment] = useState(binding.targetType === "deployment");
  const [renaming, setRenaming] = useState(false);
  const [draftName, setDraftName] = useState(binding.name);
  const [language, setLanguage] = useState("Node.js");
  const renameInput = useRef<HTMLInputElement>(null);
  useLayoutEffect(() => {
    if (renaming) {
      renameInput.current?.focus();
    }
  }, [renaming]);
  const update = trpc.appBinding.update.useMutation({
    onSuccess: async (result) => {
      if (result.name !== binding.name) {
        toast.success("Binding renamed", {
          description: `Update your code and redeploy ${appName} to use ${bindingHostVariable(result.name)}.`,
        });
      }
      setRenaming(false);
      await onSaved();
    },
    onError: (error) => toast.error(error.message),
  });
  const targetName = binding.targetAppName;
  const rule = ruleOf(binding);
  const automaticProduction = isProduction(environment);
  const targetEnvironments = targets.environments.filter(
    (env) => env.appId === binding.targetAppId,
  );
  const productionEnvironment = targetEnvironments.find(isProduction);
  const targetDeployments = targets.deployments.filter(
    (item) => item.appId === binding.targetAppId,
  );
  const description = describeTarget({
    binding,
    targetName,
    callerEnvironment: environment,
    environments: targets.environments,
    deployments: targets.deployments,
  });
  const save = (next: TargetRule, name?: string) =>
    update.mutate({ projectId, id: binding.id, ...next, ...(name ? { name } : {}) });
  const selectValue =
    pickingDeployment || binding.targetType === "deployment"
      ? "deployment"
      : binding.targetType === "environment"
        ? automaticProduction && binding.targetEnvironmentId === productionEnvironment?.id
          ? "automatic"
          : `env:${binding.targetEnvironmentId}`
        : "automatic";
  const nameCheck = bindingNameSchema.safeParse(draftName);
  const variable = bindingHostVariable(binding.name);
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
        <Button variant="ghost" size="icon" aria-label="Close binding details" onClick={onClose}>
          <IconXmarkOutline18 className="size-4" />
        </Button>
      </div>
      <div className="space-y-3 p-4">
        <h3 className="text-xs font-medium">Use in {appName}</h3>
        <dl className="divide-y divide-gray-4 rounded-lg border border-gray-4">
          {[
            { label: "Variable", value: variable },
            { label: "Hostname", value: bindingHost(binding.name) },
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
            aria-label={`Target for ${binding.name}`}
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
            {binding.targetType === "environment" &&
              !targetEnvironments.some((env) => env.id === binding.targetEnvironmentId) && (
                <option value={`env:${binding.targetEnvironmentId}`} disabled>
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
            {(targetDeployments.length > 0 || binding.targetType === "deployment") && (
              <option value="deployment">Specific deployment…</option>
            )}
          </select>
        </label>
        {pickingDeployment && (
          <select
            aria-label={`Deployment for ${binding.name}`}
            className="h-7 w-full rounded-md border border-gray-5 bg-background px-2 text-xs"
            value={binding.targetType === "deployment" ? (binding.targetDeploymentId ?? "") : ""}
            disabled={update.isLoading}
            onChange={(event) => {
              if (event.target.value) {
                save({ targetType: "deployment", targetDeploymentId: event.target.value });
              }
            }}
          >
            <option value="">Choose a deployment</option>
            {binding.targetDeploymentId &&
              !targetDeployments.some((item) => item.id === binding.targetDeploymentId) && (
                <option value={binding.targetDeploymentId} disabled>
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
                aria-label={`Name for ${binding.name}`}
                className="h-7 min-w-0 flex-1 rounded-md border border-gray-5 bg-background px-2 font-mono text-xs"
                value={draftName}
                onChange={(event) => setDraftName(event.target.value)}
                onKeyDown={(event) => {
                  if (event.key === "Escape") {
                    setRenaming(false);
                    setDraftName(binding.name);
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
                  setDraftName(binding.name);
                  setRenaming(true);
                }}
              >
                <IconPenWriting3Outline18 className="size-4" />
                Rename
              </Button>
              <Button
                variant="ghost"
                size="sm"
                aria-label={`Delete ${binding.name} binding`}
                disabled={update.isLoading}
                onClick={onDelete}
              >
                <IconTrashOutline18 className="size-4" />
                Delete
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

function BindingWires({
  root,
  highlightedId,
}: {
  root: HTMLDivElement | null;
  highlightedId: string | null;
}) {
  const [wires, setWires] = useState<Wire[]>([]);
  const markerId = useId();

  useLayoutEffect(() => {
    if (!root) {
      setWires([]);
      return;
    }
    const measure = () => {
      const from = root.querySelector<HTMLElement>("[data-wire-from]");
      if (!from) {
        setWires([]);
        return;
      }
      const base = root.getBoundingClientRect();
      const source = from.getBoundingClientRect();
      const path = (target: DOMRect) => {
        const x2 = target.left - base.left - 4;
        const y2 = target.top - base.top + target.height / 2;
        if (source.right > target.left) {
          const x1 = source.left - base.left + 12;
          const y1 = source.bottom - base.top;
          return `M ${x1} ${y1} V ${y2} H ${x2}`;
        }
        const x1 = source.right - base.left;
        const y1 = source.top - base.top + source.height / 2;
        const mid = Math.round((x1 + x2) / 2);
        return `M ${x1} ${y1} H ${mid} V ${y2} H ${x2}`;
      };
      const next: Wire[] = [];
      for (const element of root.querySelectorAll<HTMLElement>("[data-wire-to]")) {
        const id = element.dataset.wireTo;
        if (id) {
          next.push({ id, d: path(element.getBoundingClientRect()) });
        }
      }
      setWires(next);
    };
    measure();
    const observer = new ResizeObserver(measure);
    observer.observe(root);
    for (const child of root.querySelectorAll("[data-wire-to], [data-wire-from]")) {
      observer.observe(child);
    }
    return () => observer.disconnect();
  }, [root]);

  return (
    <svg
      aria-hidden="true"
      className="pointer-events-none absolute inset-0 z-10 h-full w-full overflow-visible"
    >
      <defs>
        <marker
          id={markerId}
          viewBox="0 0 6 6"
          refX="5"
          refY="3"
          markerWidth="6"
          markerHeight="6"
          orient="auto-start-reverse"
        >
          <path d="M 0 0 L 5 3 L 0 6" className="stroke-gray-9" fill="none" />
        </marker>
      </defs>
      <g
        fill="none"
        strokeLinejoin="round"
        strokeLinecap="round"
        strokeWidth={1}
        strokeDasharray="1 4"
      >
        {wires.map((wire) => (
          <path
            key={wire.id}
            d={wire.d}
            markerEnd={`url(#${markerId})`}
            data-binding-edge={wire.id}
            className={highlightedId === wire.id ? "stroke-gray-11" : "stroke-gray-8"}
          />
        ))}
      </g>
    </svg>
  );
}
