"use client";

import { collection } from "@/lib/collections";
import { trpc } from "@/lib/trpc/client";
import { mergeDeploymentTargets } from "@/lib/trpc/routers/deploy/app-connection/target-pagination";
import { connectionHostVariable } from "@/lib/trpc/routers/deploy/app-connection/validation";
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
import { AddConnectionPicker } from "./add-connection-picker";
import { ConnectionDetails } from "./connection-details";
import { ConnectionNode, Group } from "./connection-node";
import { type Environment, isProduction } from "./connection-rules";
import { ConnectionWires } from "./connection-wires";

export function ConnectionsHeader({ children }: { children?: ReactNode }) {
  return (
    <PageHeader>
      <PageHeaderContent>
        <PageHeaderTitle>Connections</PageHeaderTitle>
        <PageHeaderDescription>Connect this app to its required services.</PageHeaderDescription>
      </PageHeaderContent>
      {children && <PageHeaderActions>{children}</PageHeaderActions>}
    </PageHeader>
  );
}

export function ConnectionCanvas({
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
  const connections = trpc.appConnection.list.useQuery({
    projectId,
    appId,
    environmentId: environment.id,
  });
  const [removingId, setRemovingId] = useState<string | null>(null);
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const [search, setSearch] = useState("");
  const [canvasRoot, setCanvasRoot] = useState<HTMLDivElement | null>(null);
  const [pickerOpen, setPickerOpen] = useState(false);
  const selectedConnection = connections.data?.find((connection) => connection.id === selectedId);
  const targets = trpc.appConnection.targets.useInfiniteQuery(
    { projectId, appId, targetAppId: selectedConnection?.targetAppId },
    { getNextPageParam: (page) => page.nextCursor ?? undefined },
  );
  const refresh = async () => {
    await Promise.all([
      utils.appConnection.list.invalidate({ projectId }),
      utils.appConnection.targets.invalidate({ projectId, appId }),
    ]);
  };
  const create = trpc.appConnection.create.useMutation({
    onSuccess: async (result) => {
      setPickerOpen(false);
      setSearch("");
      await refresh();
      setSelectedId(result.id);
      toast.success("Connection created", {
        description: `Redeploy ${appQuery.data?.[0]?.name ?? "this app"} to add ${connectionHostVariable(result.name)}.`,
      });
    },
    onError: (error) => toast.error(error.message),
  });
  const remove = trpc.appConnection.delete.useMutation({
    onSuccess: async () => {
      setRemovingId(null);
      setSelectedId(null);
      await refresh();
      toast.success("Connection removed");
    },
    onError: (error) => toast.error(error.message),
  });
  if (connections.isLoading || targets.isLoading || appQuery.isLoading) {
    return (
      <>
        <ConnectionsHeader />
        <PageBody>
          <Skeleton aria-label="Loading connections" className="h-96 w-full" />
        </PageBody>
      </>
    );
  }
  if (connections.isError || targets.isError || appQuery.isError || !appQuery.data[0]) {
    return (
      <>
        <ConnectionsHeader />
        <PageBody>
          <div role="alert" className="rounded-xl border border-gray-4 p-8 text-sm text-error-11">
            We couldn't load app connections. Reload this page to try again.
          </div>
        </PageBody>
      </>
    );
  }

  const app = appQuery.data[0];
  const connected = new Set(connections.data.map((connection) => connection.targetAppId));
  const targetData = {
    ...targets.data.pages[0],
    deployments: mergeDeploymentTargets(
      [],
      targets.data.pages.flatMap((page) => page.deployments),
    ),
  };
  const others = targetData.apps.filter((target) => !connected.has(target.id));
  const removing = connections.data.find((connection) => connection.id === removingId);
  const selected = connections.data.find((connection) => connection.id === selectedId);
  const query = search.trim().toLowerCase();
  const visible = connections.data.filter((connection) =>
    `${connection.targetAppName} ${connection.name} ${connectionHostVariable(connection.name)}`
      .toLowerCase()
      .includes(query),
  );

  return (
    <>
      <ConnectionsHeader>
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
      </ConnectionsHeader>
      <PageBody>
        <div
          className={cn(
            "grid min-w-0 items-start gap-5",
            selected && "lg:grid-cols-[minmax(0,1fr)_320px]",
          )}
        >
          <div
            className="@container min-w-0 rounded-xl border border-gray-4 bg-gray-3/40"
            aria-label="Connection canvas"
          >
            <div
              ref={setCanvasRoot}
              className="relative flex min-h-52 flex-col items-start gap-8 p-5 @min-[460px]:flex-row @min-[460px]:gap-10"
            >
              <button
                type="button"
                aria-label="Clear connection selection"
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
                  connections.data.length
                    ? `Connects to · ${connections.data.length}`
                    : "Connects to"
                }
                className="w-full pl-6 @min-[460px]:max-w-sm @min-[460px]:flex-1 @min-[460px]:pl-0"
              >
                {(connections.data.length > 8 || search.length > 0) && (
                  <InputGroup className="h-8">
                    <InputGroupAddon>
                      <IconMagnifierOutline18 className="size-4" />
                    </InputGroupAddon>
                    <InputGroupInput
                      aria-label="Filter connections"
                      placeholder="Filter connections…"
                      value={search}
                      onChange={(event) => setSearch(event.target.value)}
                    />
                  </InputGroup>
                )}
                {visible.map((connection) => (
                  <ConnectionNode
                    key={connection.id}
                    connection={connection}
                    environment={environment}
                    targets={targetData}
                    selected={selected?.id === connection.id}
                    onSelect={() =>
                      setSelectedId(selected?.id === connection.id ? null : connection.id)
                    }
                  />
                ))}
                {connections.data.length === 0 && (
                  <p className="py-2 text-sm text-gray-9">
                    No connections yet. Add an app that {app.name} needs to call.
                  </p>
                )}
                {connections.data.length > 0 && visible.length === 0 && (
                  <p className="py-2 text-sm text-gray-9">No matching connections.</p>
                )}
                <AddConnectionPicker
                  open={pickerOpen}
                  onOpenChange={setPickerOpen}
                  apps={others}
                  environments={targetData.environments}
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
              <ConnectionWires
                key={visible.map((connection) => connection.id).join(",")}
                root={canvasRoot}
                highlightedId={selected?.id ?? null}
              />
            </div>
          </div>
          {selected && (
            <aside
              id="connection-details"
              aria-label="Connection details"
              className="min-w-0 break-words rounded-xl border border-gray-4 bg-background"
            >
              <ConnectionDetails
                key={selected.id}
                projectId={projectId}
                appName={app.name}
                connection={selected}
                environment={environment}
                targets={targetData}
                hasMoreDeployments={!!targets.hasNextPage}
                loadingMoreDeployments={targets.isFetchingNextPage}
                onLoadMoreDeployments={() => void targets.fetchNextPage()}
                onClose={() => {
                  canvasRoot?.querySelector<HTMLElement>('[aria-pressed="true"]')?.focus();
                  setSelectedId(null);
                }}
                onRemove={() => setRemovingId(selected.id)}
                onSaved={refresh}
              />
            </aside>
          )}
        </div>
      </PageBody>
      <DialogContainer
        isOpen={!!removing}
        onOpenChange={(open) => {
          if (!open) {
            setRemovingId(null);
          }
        }}
        title={`Remove the ${removing?.targetAppName ?? ""} connection?`}
        subTitle={
          removing
            ? `${app.name} can no longer connect to ${removing.targetAppName} once the change applies. ${connectionHostVariable(removing.name)} stays in running deployments until you redeploy ${app.name}.`
            : undefined
        }
        footer={
          <div className="flex gap-2">
            <Button variant="outline" onClick={() => setRemovingId(null)}>
              Keep connection
            </Button>
            <Button
              variant="primary"
              color="danger"
              disabled={remove.isLoading}
              onClick={() => {
                if (removing) {
                  remove.mutate({ projectId, id: removing.id });
                }
              }}
            >
              Remove connection
            </Button>
          </div>
        }
      />
    </>
  );
}
