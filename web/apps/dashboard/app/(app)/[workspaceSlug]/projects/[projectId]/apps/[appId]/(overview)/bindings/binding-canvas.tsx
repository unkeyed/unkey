"use client";

import { collection } from "@/lib/collections";
import { trpc } from "@/lib/trpc/client";
import { bindingHostVariable } from "@/lib/trpc/routers/deploy/app-binding/validation";
import { and, eq, useLiveQuery } from "@tanstack/react-db";
import { IconCubeOutline18, IconLinkOutline18, IconMagnifierOutline18 } from "@unkey/icons";
import {
  Button,
  DialogContainer,
  InputGroup,
  InputGroupAddon,
  InputGroupInput,
  PageBody,
  PageHeader,
  PageHeaderActions,
  PageHeaderContent,
  PageHeaderDescription,
  PageHeaderTitle,
  Skeleton,
  cn,
  toast,
} from "@unkey/ui";
import { type ReactNode, useState } from "react";
import { AddBindingPicker } from "./add-binding-picker";
import { BindingDetails } from "./binding-details";
import { BindingNode, Group } from "./binding-node";
import { type Environment, isProduction } from "./binding-rules";
import { BindingWires } from "./binding-wires";

export function BindingsHeader({ children }: { children?: ReactNode }) {
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

export function BindingCanvas({
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
