"use client";

import { Collapsible, CollapsibleContent, CollapsibleTrigger } from "@/components/ui/collapsible";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { routes } from "@/lib/navigation/routes";
import { trpc } from "@/lib/trpc/client";
import { ChevronDown, Earth, Plus, Trash, TriangleWarning2 } from "@unkey/icons";
import {
  AlertBanner,
  AlertBannerTitle,
  Button,
  CopyButton,
  DialogContainer,
  Empty,
  Input,
  PageBody,
  PageContainer,
  PageHeader,
  PageHeaderActions,
  PageHeaderContent,
  PageHeaderTitle,
  SettingCard,
  SettingCardGroup,
  SettingsDangerZone,
  TimestampInfo,
  toast,
} from "@unkey/ui";
import { useRouter } from "next/navigation";
import { use, useEffect, useRef, useState } from "react";
import { AxiomLogo } from "../axiom-logo";
import {
  type HeaderField,
  emptyHeader,
  headerFieldsSchema,
  toHeaderRecord,
} from "../header-fields";
import { type DrainStatus, StatusBadge } from "../logdrain-ui";
import { DeliveryOverview } from "./charts";

type DrainBase = {
  id: string;
  name: string;
  status: DrainStatus;
};
type Drain = DrainBase &
  (
    | {
        kind: "http";
        config: { url: string; format: "json" | "ndjson"; headers: HeaderField[] };
      }
    | { kind: "axiom"; config: { dataset: string } }
  );

function getDestination(drain: Drain) {
  return drain.kind === "axiom" ? drain.config.dataset : drain.config.url;
}

type EditableHeader = HeaderField & { id: number };

function editableHeaders(headers: HeaderField[], firstId: number): EditableHeader[] {
  const fields = headers.length > 0 ? headers : [{ ...emptyHeader }];
  return fields.map((field, index) => ({ id: firstId + index, ...field }));
}

type RecentError = {
  time: number;
  outcome: "error" | "transient_error" | "permanent_error";
  responseStatus: number;
  responseBody: string;
  error: string;
};

export default function LogdrainDetailPage(props: { params: Promise<{ drainId: string }> }) {
  const { drainId } = use(props.params);
  const query = trpc.logdrain.get.useQuery({ id: drainId });
  const metrics = trpc.logdrain.metrics.useQuery({ drainId, hours: 24 });
  const recentErrors = trpc.logdrain.recentErrors.useQuery({ drainId });
  const drain = query.data;

  if (query.isLoading) {
    return <DetailSkeleton />;
  }
  if (query.isError) {
    return (
      <PageContainer>
        <PageBody>
          <Empty>
            <Empty.Title>Unable to load log drain</Empty.Title>
            <Empty.Description>{query.error.message}</Empty.Description>
          </Empty>
        </PageBody>
      </PageContainer>
    );
  }
  if (!drain) {
    return (
      <PageContainer>
        <PageBody>
          <Empty>
            <Empty.Title>Log drain not found</Empty.Title>
            <Empty.Description>It may have been deleted.</Empty.Description>
          </Empty>
        </PageBody>
      </PageContainer>
    );
  }
  return (
    <Detail
      drain={drain}
      metricsSeries={metrics.data?.series}
      metricsLoading={metrics.isLoading}
      metricsError={metrics.isError}
      recentErrorEntries={recentErrors.data}
      recentErrorsLoading={recentErrors.isLoading}
      recentErrorsError={recentErrors.isError}
    />
  );
}

function Detail({
  drain,
  metricsSeries,
  metricsLoading,
  metricsError,
  recentErrorEntries,
  recentErrorsLoading,
  recentErrorsError,
}: {
  drain: Drain;
  metricsSeries?: Parameters<typeof DeliveryOverview>[0]["series"];
  metricsLoading: boolean;
  metricsError: boolean;
  recentErrorEntries?: RecentError[];
  recentErrorsLoading: boolean;
  recentErrorsError: boolean;
}) {
  const [name, setName] = useState(drain.name);
  const [destination, setDestination] = useState(getDestination(drain));
  const configuredHeaders = drain.kind === "http" ? drain.config.headers : [];
  const nextHeaderId = useRef(Math.max(configuredHeaders.length, 1));
  const [headers, setHeaders] = useState<EditableHeader[]>(() =>
    editableHeaders(configuredHeaders, 0),
  );
  const [token, setToken] = useState("");
  const [confirmDelete, setConfirmDelete] = useState(false);
  const utils = trpc.useUtils();
  const router = useRouter();
  const workspace = useWorkspaceNavigation();
  const update = trpc.logdrain.update.useMutation({
    onSuccess: () => {
      utils.logdrain.list.invalidate();
      utils.logdrain.get.invalidate({ id: drain.id });
      toast.success("Log drain updated");
    },
    onError: (error) => toast.error(error.message),
  });
  const remove = trpc.logdrain.delete.useMutation({
    onSuccess: () => {
      utils.logdrain.list.invalidate();
      toast.success("Log drain deleted");
      router.push(routes.settings.logdrains.list({ workspaceSlug: workspace.slug }));
    },
    onError: (error) => toast.error(error.message),
  });

  useEffect(() => setName(drain.name), [drain.name]);
  useEffect(() => setDestination(getDestination(drain)), [drain]);
  useEffect(() => {
    if (drain.kind === "http") {
      const fields = editableHeaders(drain.config.headers, nextHeaderId.current);
      nextHeaderId.current += fields.length;
      setHeaders(fields);
    }
  }, [drain]);

  const currentDestination = getDestination(drain);
  const saveDestination = () => {
    const value = destination.trim();
    if (drain.kind === "http") {
      update.mutate({ id: drain.id, destination: { kind: "http", config: { url: value } } });
    } else if (drain.kind === "axiom") {
      update.mutate({ id: drain.id, destination: { kind: "axiom", config: { dataset: value } } });
    }
  };
  const saveHeaders = () => {
    const parsed = headerFieldsSchema.safeParse(headers);
    if (!parsed.success) {
      toast.error(parsed.error.issues[0]?.message ?? "Check the header fields");
      return;
    }
    if (drain.kind === "http") {
      update.mutate({
        id: drain.id,
        destination: {
          kind: "http",
          config: { url: currentDestination, headers: toHeaderRecord(parsed.data) },
        },
      });
    }
  };
  const saveToken = () => {
    if (drain.kind === "axiom") {
      update.mutate({
        id: drain.id,
        destination: {
          kind: "axiom",
          config: { dataset: currentDestination, token: token.trim() },
        },
      });
    }
  };

  return (
    <PageContainer>
      <PageHeader>
        <PageHeaderContent>
          <div className="flex items-center gap-3">
            <PageHeaderTitle>{drain.name}</PageHeaderTitle>
            <StatusBadge status={drain.status} />
          </div>
        </PageHeaderContent>
        <PageHeaderActions>
          <span className="flex items-center gap-2 whitespace-nowrap text-[13px] font-medium text-accent-12">
            <span className="flex size-5 shrink-0 items-center justify-center rounded-sm bg-grayA-3 text-gray-11">
              {drain.kind === "axiom" ? (
                <AxiomLogo className="size-3.5" />
              ) : (
                <Earth iconSize="sm-regular" />
              )}
            </span>
            {drain.kind === "axiom" ? "Axiom" : "HTTP"}
          </span>
        </PageHeaderActions>
      </PageHeader>
      <PageBody>
        <DeliveryOverview series={metricsSeries} loading={metricsLoading} error={metricsError} />
        <RecentDeliveryErrors
          entries={recentErrorEntries}
          loading={recentErrorsLoading}
          error={recentErrorsError}
        />
        <section className="flex flex-col gap-3">
          <h2 className="text-sm font-medium text-accent-12">Settings</h2>
          <SettingCardGroup>
            <SettingCard
              title="Drain name"
              description="A human-readable name for this log drain."
              contentWidth="w-full lg:w-[420px] justify-end"
            >
              <Input value={name} onChange={(event) => setName(event.target.value)} />
              <Button
                variant="primary"
                loading={update.isLoading}
                disabled={!name.trim() || name.trim() === drain.name}
                onClick={() => update.mutate({ id: drain.id, name: name.trim() })}
              >
                Save
              </Button>
            </SettingCard>
            <SettingCard
              title="Drain ID"
              description="The identifier for this log drain. The delivery offset is tracked against this ID."
              contentWidth="w-full lg:w-[420px] justify-end"
            >
              <div className="flex flex-row justify-end items-center">
                <div className="flex flex-row justify-between min-w-[327px] pl-2 pr-2 py-2 bg-gray-2 dark:bg-black border rounded-lg border-grayA-5">
                  <div className="text-sm text-gray-11">{drain.id}</div>
                  <CopyButton value={drain.id} variant="ghost" toastMessage={drain.id} />
                </div>
              </div>
            </SettingCard>
            <SettingCard
              title={drain.kind === "axiom" ? "Dataset" : "HTTPS endpoint"}
              description="Where deliveries are sent. Changing this keeps the drain ID and its delivery offset."
              contentWidth="w-full lg:w-[420px] justify-end"
            >
              <Input value={destination} onChange={(event) => setDestination(event.target.value)} />
              <Button
                variant="primary"
                loading={update.isLoading}
                disabled={!destination.trim() || destination.trim() === currentDestination}
                onClick={saveDestination}
              >
                Save
              </Button>
            </SettingCard>
            {drain.kind === "http" ? (
              <SettingCard
                title="Headers"
                description="Add optional headers for each delivery. Saving replaces the current headers."
                contentWidth="w-full lg:w-[520px] justify-end"
              >
                <div className="flex w-full flex-col gap-3">
                  {headers.map((header, index) => (
                    <div key={header.id} className="flex items-center gap-2">
                      <Input
                        aria-label={`Header ${index + 1} name`}
                        placeholder="Header name"
                        value={header.name}
                        onChange={(event) =>
                          setHeaders((current) =>
                            current.map((item, itemIndex) =>
                              itemIndex === index ? { ...item, name: event.target.value } : item,
                            ),
                          )
                        }
                      />
                      <Input
                        aria-label={`Header ${index + 1} value`}
                        placeholder="Header value"
                        value={header.value}
                        onChange={(event) =>
                          setHeaders((current) =>
                            current.map((item, itemIndex) =>
                              itemIndex === index ? { ...item, value: event.target.value } : item,
                            ),
                          )
                        }
                      />
                      {headers.length > 1 ? (
                        <Button
                          variant="ghost"
                          size="sm"
                          className="size-9 shrink-0 justify-center px-0 text-gray-11"
                          aria-label={`Remove header ${index + 1}`}
                          onClick={() =>
                            setHeaders((current) =>
                              current.filter((_, itemIndex) => itemIndex !== index),
                            )
                          }
                        >
                          <Trash iconSize="sm-regular" />
                        </Button>
                      ) : null}
                    </div>
                  ))}
                  <div className="flex justify-between gap-2">
                    <Button
                      variant="outline"
                      disabled={headers.length >= 32}
                      onClick={() => {
                        const id = nextHeaderId.current;
                        nextHeaderId.current += 1;
                        setHeaders((current) => [...current, { id, ...emptyHeader }]);
                      }}
                    >
                      <Plus iconSize="sm-regular" />
                      Add header
                    </Button>
                    <Button variant="primary" loading={update.isLoading} onClick={saveHeaders}>
                      Save
                    </Button>
                  </div>
                </div>
              </SettingCard>
            ) : (
              <SettingCard
                title="Token"
                description="Rotate the token used to authenticate against the destination. The current token stays active until you save a new one."
                contentWidth="w-full lg:w-[420px] justify-end"
              >
                <Input
                  type="password"
                  value={token}
                  placeholder="Enter a new token"
                  onChange={(event) => setToken(event.target.value)}
                />
                <Button
                  variant="primary"
                  loading={update.isLoading}
                  disabled={!token.trim()}
                  onClick={saveToken}
                >
                  Save
                </Button>
              </SettingCard>
            )}
            {drain.kind === "http" && (
              <SettingCard
                title="Body format"
                description="JSON array sends one array of event objects. NDJSON sends one JSON object per line."
                contentWidth="w-full lg:w-[420px] justify-end"
              >
                <div className="flex rounded-lg border border-grayA-4 p-1">
                  {(["json", "ndjson"] as const).map((value) => (
                    <Button
                      key={value}
                      size="sm"
                      variant={drain.config.format === value ? "primary" : "ghost"}
                      loading={update.isLoading}
                      onClick={() => {
                        if (drain.config.format !== value) {
                          update.mutate({
                            id: drain.id,
                            destination: {
                              kind: "http",
                              config: { url: drain.config.url, format: value },
                            },
                          });
                        }
                      }}
                    >
                      {value === "json" ? "JSON array" : "NDJSON"}
                    </Button>
                  ))}
                </div>
              </SettingCard>
            )}
            <SettingCard
              title="Status"
              description="Pause to stop deliveries. Resume to continue from the tracked offset."
              contentWidth="w-full lg:w-[420px] justify-end"
            >
              <Button
                variant="primary"
                loading={update.isLoading}
                onClick={() =>
                  update.mutate({
                    id: drain.id,
                    status: drain.status === "running" ? "paused_by_user" : "running",
                  })
                }
              >
                {drain.status === "paused_by_failure"
                  ? "Resume"
                  : drain.status === "running"
                    ? "Pause"
                    : "Resume"}
              </Button>
            </SettingCard>
          </SettingCardGroup>
        </section>
        <SettingsDangerZone>
          <SettingCard
            title="Delete log drain"
            description="Deletes this drain and stops delivery. The delivery offset is lost. This cannot be undone."
            contentWidth="w-full lg:w-[420px] justify-end"
          >
            <Button variant="destructive" onClick={() => setConfirmDelete(true)}>
              Delete log drain…
            </Button>
          </SettingCard>
        </SettingsDangerZone>
        <DialogContainer
          isOpen={confirmDelete}
          onOpenChange={setConfirmDelete}
          title={`Delete ${drain.name}?`}
          subTitle="This stops delivery immediately and cannot be undone."
          footer={
            <div className="flex w-full gap-2">
              <Button className="flex-1" variant="outline" onClick={() => setConfirmDelete(false)}>
                Cancel
              </Button>
              <Button
                className="flex-1"
                variant="primary"
                loading={remove.isLoading}
                onClick={() => remove.mutate({ id: drain.id })}
              >
                Delete log drain
              </Button>
            </div>
          }
        >
          <p className="text-sm text-gray-10">
            Existing delivery history is retained, but this destination will no longer receive audit
            logs.
          </p>
        </DialogContainer>
      </PageBody>
    </PageContainer>
  );
}

function RecentDeliveryErrors({
  entries,
  loading,
  error,
}: {
  entries?: RecentError[];
  loading: boolean;
  error: boolean;
}) {
  if (loading) {
    return null;
  }

  if (error) {
    return (
      <AlertBanner variant="error">
        <AlertBannerTitle>Could not load delivery error details</AlertBannerTitle>
      </AlertBanner>
    );
  }

  if (!entries?.length) {
    return null;
  }

  return (
    <Collapsible className="overflow-hidden rounded-xl border border-grayA-4 bg-gray-1">
      <CollapsibleTrigger className="flex w-full items-center gap-3 px-4 py-3 text-left transition-colors hover:bg-grayA-2 [&[data-panel-open]_.error-chevron]:rotate-180">
        <span className="flex size-8 shrink-0 items-center justify-center rounded-lg bg-errorA-3 text-error-11">
          <TriangleWarning2 iconSize="sm-regular" aria-hidden="true" />
        </span>
        <span className="min-w-0 flex-1">
          <span className="block text-sm font-medium text-gray-12">
            Some recent deliveries failed
          </span>
          <span className="block text-xs text-gray-10">
            {entries.length} {entries.length === 1 ? "failure" : "failures"} in the past 24 hours.
          </span>
        </span>
        <ChevronDown
          iconSize="sm-regular"
          className="error-chevron text-gray-9 transition-transform duration-200"
          aria-hidden="true"
        />
      </CollapsibleTrigger>
      <CollapsibleContent>
        <div className="overflow-x-auto border-t border-grayA-4 bg-background">
          <table className="w-full min-w-[680px] table-fixed border-collapse text-left">
            <colgroup>
              <col className="w-44" />
              <col className="w-32" />
              <col />
            </colgroup>
            <thead className="bg-grayA-2">
              <tr className="border-b border-grayA-4">
                <th className="px-4 py-2 text-xs font-medium text-gray-10">Time</th>
                <th className="px-4 py-2 text-xs font-medium text-gray-10">Status</th>
                <th className="px-4 py-2 text-xs font-medium text-gray-10">Response</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-grayA-4">
              {entries.map((entry) => {
                const detail =
                  entry.responseBody || entry.error || "The destination returned no details.";
                const status =
                  entry.responseStatus > 0
                    ? entry.responseStatus
                    : entry.outcome === "permanent_error"
                      ? "Permanent error"
                      : entry.outcome === "transient_error"
                        ? "Transient error"
                        : "Error";

                return (
                  <tr
                    key={`${entry.time}-${entry.outcome}-${entry.responseStatus}`}
                    className="align-top transition-colors hover:bg-grayA-2"
                  >
                    <td className="px-4 py-3">
                      <TimestampInfo
                        value={entry.time}
                        className="whitespace-nowrap font-mono text-gray-10 underline decoration-dotted"
                      />
                    </td>
                    <td className="px-4 py-3">
                      <span className="inline-flex rounded-md border border-errorA-4 bg-errorA-2 px-2 py-1 font-mono text-xs font-medium text-error-11">
                        {status}
                      </span>
                    </td>
                    <td className="px-4 py-3">
                      <div className="flex items-start gap-2">
                        <pre className="max-h-32 min-w-0 flex-1 overflow-auto whitespace-pre-wrap break-words font-mono text-xs leading-5 text-gray-12">
                          {formatResponseBody(detail)}
                        </pre>
                        <CopyButton
                          value={detail}
                          variant="ghost"
                          size="sm"
                          className="shrink-0"
                          aria-label={entry.responseBody ? "Copy response body" : "Copy error"}
                        />
                      </div>
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>
      </CollapsibleContent>
    </Collapsible>
  );
}

function formatResponseBody(body: string): string {
  try {
    const parsed: unknown = JSON.parse(body);
    return JSON.stringify(parsed, null, 2) ?? body;
  } catch {
    return body;
  }
}

function DetailSkeleton() {
  return (
    <PageContainer>
      <PageHeader>
        <PageHeaderContent>
          <div className="h-5 w-48 animate-pulse rounded-sm bg-grayA-3" />
        </PageHeaderContent>
      </PageHeader>
      <PageBody aria-busy="true">
        <div className="h-52 animate-pulse rounded-xl bg-grayA-3" />
        <div className="h-96 animate-pulse rounded-xl bg-grayA-3" />
      </PageBody>
    </PageContainer>
  );
}
