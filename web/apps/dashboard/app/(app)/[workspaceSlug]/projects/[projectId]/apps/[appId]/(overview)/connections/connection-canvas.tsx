"use client";

import { collection } from "@/lib/collections";
import { DEPLOYMENT_STATUSES } from "@/lib/collections/deploy/deployment-status";
import { showSettingsBanner } from "@/lib/collections/deploy/pending-redeploy";
import { queryKeys } from "@/lib/query-keys";
import { getErrorMessage, getUnkeyClient } from "@/lib/unkey-client";
import { and, eq, inArray, useLiveQuery } from "@tanstack/react-db";
import { useInfiniteQuery, useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import type { V2ConnectionsCreateConnectionRequestBody } from "@unkey/api/models/components";
import {
  IconChevronDownOutline12,
  IconCloudOutline18,
  IconCubeOutline18,
  IconEyeOutline18,
  IconMagnifierOutline18,
} from "@unkey/icons";
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
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
  Skeleton,
  cn,
  toast,
} from "@unkey/ui";
import { type ReactNode, useState } from "react";
import { isRedeployableDeploymentStatus } from "../deployments/components/table/components/actions/deployment-action-eligibility";
import { AddConnectionPicker } from "./add-connection-picker";
import { ConnectionDetails } from "./connection-details";
import { ConnectionNode, Group } from "./connection-node";
import {
  type Environment,
  connectionHostVariable,
  isProduction,
  mergeDeploymentTargets,
} from "./connection-rules";
import { ConnectionWires } from "./connection-wires";
import { HowConnectionsWorkPanel } from "./how-connections-work";

function EnvironmentLabel({ environment }: { environment: Environment }) {
  const Icon = isProduction(environment) ? IconCloudOutline18 : IconEyeOutline18;
  return (
    <span className="inline-flex items-center gap-1 whitespace-nowrap font-mono text-sm text-gray-12">
      <Icon className="size-3.5 shrink-0" />
      <span className="capitalize">{environment.slug}</span>
    </span>
  );
}

export function ConnectionsHeader({ children }: { children?: ReactNode }) {
  const [guideOpen, setGuideOpen] = useState(false);

  return (
    <>
      <PageHeader>
        <PageHeaderContent>
          <PageHeaderTitle>Connections</PageHeaderTitle>
          <PageHeaderDescription>Connect this app to its required services.</PageHeaderDescription>
        </PageHeaderContent>
        <PageHeaderActions className="max-w-full">
          <Button size="md" variant="outline" onClick={() => setGuideOpen(true)}>
            How connections work
          </Button>
          {children}
        </PageHeaderActions>
      </PageHeader>
      <HowConnectionsWorkPanel isOpen={guideOpen} onClose={() => setGuideOpen(false)} />
    </>
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
  const queryClient = useQueryClient();
  const appQuery = useLiveQuery(
    (q) =>
      q
        .from({ app: collection.apps })
        .where(({ app }) => and(eq(app.projectId, projectId), eq(app.id, appId))),
    [projectId, appId],
  );
  const currentDeploymentId = appQuery.data?.[0]?.currentDeploymentId;
  const redeploySource = useLiveQuery(
    (q) => {
      if (isProduction(environment) && !currentDeploymentId) {
        return null;
      }
      let query = q
        .from({ deployment: collection.deployments })
        .where(({ deployment }) =>
          and(
            eq(deployment.projectId, projectId),
            eq(deployment.appId, appId),
            eq(deployment.environmentId, environment.id),
            inArray(deployment.status, DEPLOYMENT_STATUSES.filter(isRedeployableDeploymentStatus)),
          ),
        );
      if (isProduction(environment) && currentDeploymentId) {
        query = query.where(({ deployment }) => eq(deployment.id, currentDeploymentId));
      }
      return query.orderBy(({ deployment }) => deployment.createdAt, "desc").limit(1);
    },
    [projectId, appId, environment.id, environment.kind, currentDeploymentId],
  );
  const deployment = redeploySource.data?.[0];
  const showRedeployBanner = () => {
    if (deployment) {
      showSettingsBanner({ deployment, environmentSlug: environment.slug });
    }
  };
  const connections = useQuery({
    queryKey: queryKeys.connections.list(projectId, appId, environment.id),
    queryFn: async ({ signal }) => {
      const firstPage = await getUnkeyClient().connections.listConnections(
        { project: projectId, app: appId, environment: environment.id },
        { signal },
      );
      const pages = await Array.fromAsync(firstPage);
      return pages.flatMap((page) => page.result.data).sort((a, b) => a.name.localeCompare(b.name));
    },
  });
  const retained = useQuery({
    queryKey: queryKeys.connections.retained(projectId, appId, environment.id),
    queryFn: async ({ signal }) =>
      (
        await getUnkeyClient().connections.listRetainedConnections(
          { project: projectId, app: appId, environment: environment.id },
          { signal },
        )
      ).data,
  });
  const [removingId, setRemovingId] = useState<string | null>(null);
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const [highlightedId, setHighlightedId] = useState<string | null>(null);
  const [search, setSearch] = useState("");
  const [canvasRoot, setCanvasRoot] = useState<HTMLDivElement | null>(null);
  const [pickerOpen, setPickerOpen] = useState(false);
  const selectedConnection = connections.data?.find((connection) => connection.id === selectedId);
  const targets = useInfiniteQuery({
    queryKey: queryKeys.connections.targets(
      projectId,
      appId,
      environment.id,
      selectedConnection?.targetAppId,
    ),
    queryFn: ({ pageParam, signal }: { pageParam?: string; signal?: AbortSignal }) =>
      getUnkeyClient().connections.listTargets(
        {
          project: projectId,
          app: appId,
          environment: environment.id,
          targetAppId: selectedConnection?.targetAppId,
          cursor: pageParam,
        },
        { signal },
      ),
    getNextPageParam: (page) =>
      page.pagination.hasMore ? (page.pagination.cursor ?? undefined) : undefined,
    keepPreviousData: true,
  });
  const refresh = async () => {
    await queryClient.invalidateQueries({ queryKey: queryKeys.connections.app(projectId, appId) });
  };
  const create = useMutation({
    mutationFn: async (request: V2ConnectionsCreateConnectionRequestBody) =>
      (await getUnkeyClient().connections.createConnection(request)).data,
    onSuccess: async (result) => {
      setPickerOpen(false);
      setSearch("");
      showRedeployBanner();
      toast.success("Connection created");
      await refresh();
      setSelectedId(result.id);
    },
    onError: (error) => toast.error(getErrorMessage(error)),
  });
  const remove = useMutation({
    mutationFn: (connectionId: string) =>
      getUnkeyClient().connections.deleteConnection({ connectionId }),
    onSuccess: async () => {
      setRemovingId(null);
      setSelectedId(null);
      showRedeployBanner();
      toast.success("Connection removed");
      await refresh();
    },
    onError: (error) => toast.error(getErrorMessage(error)),
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
    ...targets.data.pages[0].data,
    deployments: mergeDeploymentTargets(
      [],
      targets.data.pages.flatMap((page) => page.data.deployments),
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
        <Select
          value={environment.id}
          items={environments.map((env) => ({ value: env.id, label: env.slug }))}
          onValueChange={(value) => {
            if (value !== null) {
              onEnvironmentChange(value);
            }
          }}
        >
          <SelectTrigger
            aria-label="Caller environment"
            size="sm"
            wrapperClassName="w-44"
            rightIcon={<IconChevronDownOutline12 className="absolute right-3 opacity-70" />}
          >
            <SelectValue>
              <EnvironmentLabel environment={environment} />
            </SelectValue>
          </SelectTrigger>
          <SelectContent align="end">
            {environments.map((env) => (
              <SelectItem key={env.id} value={env.id}>
                <EnvironmentLabel environment={env} />
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </ConnectionsHeader>
      <PageBody>
        <p className="text-sm text-gray-9">
          Defaults for new deployments. Existing deployments keep their saved connections.
        </p>
        <div
          className={cn(
            "grid min-w-0 items-start gap-x-0 gap-y-5 lg:grid-cols-[minmax(0,1fr)_0px] motion-safe:transition-[grid-template-columns,column-gap] motion-safe:duration-200 motion-safe:ease-out",
            selected && "lg:grid-cols-[minmax(0,1fr)_320px] lg:gap-x-5",
          )}
        >
          <div
            className="@container min-w-0 rounded-lg border border-grayA-4 bg-grayA-2"
            aria-label="Connection canvas"
          >
            <div
              ref={setCanvasRoot}
              className="relative flex min-h-52 flex-col items-start gap-10 p-6 @min-[520px]:flex-row @min-[520px]:justify-center @min-[520px]:gap-16"
            >
              <button
                type="button"
                aria-label="Clear connection selection"
                className="absolute inset-0 rounded-lg"
                tabIndex={-1}
                disabled={!selected}
                onClick={() => setSelectedId(null)}
              />
              <Group label="This app" className="w-full @min-[520px]:w-44">
                <div
                  data-wire-from
                  className="overflow-hidden rounded-lg bg-raised shadow-sm ring-1 ring-grayA-5"
                >
                  <span className="flex min-w-0 items-center gap-2.5 px-3 py-3">
                    <span className="flex size-7 shrink-0 items-center justify-center rounded-md bg-grayA-3 text-gray-11">
                      <IconCubeOutline18 className="size-4" />
                    </span>
                    <span className="truncate text-sm font-medium text-gray-12" title={app.name}>
                      {app.name}
                    </span>
                  </span>
                  <div
                    className="truncate border-t border-grayA-4 bg-grayA-2 px-3 py-2 font-mono text-2xs text-gray-10"
                    title={environment.slug}
                  >
                    {environment.slug}
                  </div>
                </div>
              </Group>
              <Group
                label={
                  connections.data.length
                    ? `Connects to · ${connections.data.length}`
                    : "Connects to"
                }
                className="w-full pl-8 @min-[520px]:max-w-80 @min-[520px]:flex-1 @min-[520px]:pl-0"
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
                    onHighlight={setHighlightedId}
                  />
                ))}
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
                      project: projectId,
                      app: appId,
                      environment: environment.id,
                      targetApp: targetAppId,
                      ...rule,
                    })
                  }
                />
              </Group>
              <ConnectionWires
                key={visible.map((connection) => connection.id).join(",")}
                root={canvasRoot}
                highlightedId={highlightedId ?? selected?.id ?? null}
              />
            </div>
          </div>
          {selected && (
            <aside
              id="connection-details"
              aria-label="Connection details"
              className="min-w-0 overflow-hidden animate-in fade-in duration-200 motion-reduce:animate-none"
            >
              <div className="min-w-0 break-words rounded-xl border border-gray-4 bg-background lg:w-80">
                <ConnectionDetails
                  key={selected.id}
                  appName={app.name}
                  connection={selected}
                  environment={environment}
                  targets={targetData}
                  hasMoreDeployments={!targets.isPreviousData && !!targets.hasNextPage}
                  loadingMoreDeployments={targets.isFetchingNextPage}
                  onLoadMoreDeployments={() => void targets.fetchNextPage()}
                  onClose={() => {
                    canvasRoot?.querySelector<HTMLElement>('[aria-pressed="true"]')?.focus();
                    setSelectedId(null);
                  }}
                  onRemove={() => setRemovingId(selected.id)}
                  onSaved={() => {
                    showRedeployBanner();
                    return refresh();
                  }}
                />
              </div>
            </aside>
          )}
        </div>
        {retained.isError ? (
          <p role="alert" className="mt-5 text-sm text-error-11">
            We couldn't load connections saved in existing deployments. Reload to try again.
          </p>
        ) : retained.data && retained.data.length > 0 ? (
          <section className="mt-5 space-y-3 rounded-xl border border-gray-4 p-5">
            <h2 className="text-sm font-medium">Existing deployments only</h2>
            <p className="text-xs leading-5 text-gray-9">
              These connections were removed from the defaults. Saved deployments still include
              them.
            </p>
            {retained.data.map((connection) => (
              <div key={connection.id} className="flex items-center justify-between gap-3">
                <div className="min-w-0">
                  <p className="text-sm">{connection.targetAppName}</p>
                  <p className="break-all text-xs text-gray-9">
                    {connection.id} · {connection.deployments} saved{" "}
                    {connection.deployments === 1 ? "deployment" : "deployments"}
                  </p>
                </div>
              </div>
            ))}
          </section>
        ) : null}
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
            ? `New deployments of ${app.name} will not include this connection or ${connectionHostVariable(removing.name)}. Existing deployments keep their saved connection and variable.`
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
                  remove.mutate(removing.id);
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
