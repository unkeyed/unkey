"use client";

import { Collapsible, CollapsibleContent, CollapsibleTrigger } from "@/components/ui/collapsible";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { routes } from "@/lib/navigation/routes";
import { trpc } from "@/lib/trpc/client";
import { ChevronDown, Earth, Plus, Trash, TriangleWarning2 } from "@unkey/icons";
import {
  AlertBanner,
  AlertBannerActions,
  AlertBannerDescription,
  AlertBannerTitle,
  Button,
  CopyButton,
  DialogContainer,
  Empty,
  Input,
  Item,
  ItemActions,
  ItemContent,
  ItemDescription,
  ItemTitle,
  PageBody,
  PageContainer,
  PageHeader,
  PageHeaderActions,
  PageHeaderContent,
  PageHeaderTitle,
  SettingCard,
  SettingCardGroup,
  SettingsDangerZone,
  SettingsZoneRow,
  Skeleton,
  TimestampInfo,
  toast,
} from "@unkey/ui";
import { useRouter } from "next/navigation";
import { type ReactNode, use, useEffect, useRef, useState } from "react";
import { AxiomLogo } from "../axiom-logo";
import { type HeaderUpdateField, emptyHeader, headerUpdateFieldsSchema } from "../header-fields";
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
        config: { url: string; format: "json" | "ndjson"; headers: string[] };
      }
    | { kind: "axiom"; config: { dataset: string } }
  );

function getDestination(drain: Drain) {
  switch (drain.kind) {
    case "http":
      return drain.config.url;
    case "axiom":
      return drain.config.dataset;
    default:
      throw new Error(`Unsupported log drain sink: ${drain satisfies never}`);
  }
}

function getConfiguredHeaders(drain: Drain): string[] {
  switch (drain.kind) {
    case "http":
      return drain.config.headers;
    case "axiom":
      return [];
    default:
      throw new Error(`Unsupported log drain sink: ${drain satisfies never}`);
  }
}

function SinkType({ kind }: { kind: Drain["kind"] }) {
  switch (kind) {
    case "http":
      return (
        <>
          <span className="flex size-5 shrink-0 items-center justify-center rounded-sm bg-grayA-3 text-gray-11">
            <Earth iconSize="sm-regular" />
          </span>
          <span>HTTP</span>
        </>
      );
    case "axiom":
      return (
        <>
          <span className="flex size-5 shrink-0 items-center justify-center rounded-sm bg-grayA-3 text-gray-11">
            <AxiomLogo className="size-3.5" />
          </span>
          <span>Axiom</span>
        </>
      );
    default:
      throw new Error(`Unsupported log drain sink: ${kind satisfies never}`);
  }
}

function destinationCopy(kind: Drain["kind"]): {
  title: string;
  description: string;
  saveLabel: string;
} {
  switch (kind) {
    case "http":
      return {
        title: "HTTPS endpoint",
        description: "Unkey sends each audit log batch to this URL.",
        saveLabel: "Save endpoint",
      };
    case "axiom":
      return {
        title: "Dataset",
        description: "Unkey sends audit logs to this Axiom dataset.",
        saveLabel: "Save dataset",
      };
    default:
      throw new Error(`Unsupported log drain sink: ${kind satisfies never}`);
  }
}

type EditableHeader =
  | { id: number; source: "stored"; name: string; value: string }
  | { id: number; source: "new"; name: string; value: string };

function editableHeaders(headers: string[], firstId: number): EditableHeader[] {
  if (headers.length === 0) {
    return [{ id: firstId, source: "new", ...emptyHeader }];
  }
  return headers.map((name, index) => ({
    id: firstId + index,
    source: "stored",
    name,
    value: "",
  }));
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
  const configuredHeaders = getConfiguredHeaders(drain);
  const nextHeaderId = useRef(Math.max(configuredHeaders.length, 1));
  const [headers, setHeaders] = useState<EditableHeader[]>(() =>
    editableHeaders(configuredHeaders, 0),
  );
  const [token, setToken] = useState("");
  const [confirmDelete, setConfirmDelete] = useState(false);
  const destinationLabels = destinationCopy(drain.kind);
  const utils = trpc.useUtils();
  const router = useRouter();
  const workspace = useWorkspaceNavigation();
  const update = trpc.logdrain.update.useMutation({
    onSuccess: (_data, variables) => {
      switch (variables.destination?.kind) {
        case "http":
          if (variables.destination.config.headers !== undefined) {
            const fields = editableHeaders(
              variables.destination.config.headers.map((header) => header.name),
              nextHeaderId.current,
            );
            nextHeaderId.current += fields.length;
            setHeaders(fields);
          }
          break;
        case "axiom":
          if (variables.destination.config.token) {
            setToken("");
          }
          break;
        case undefined:
          break;
        default:
          throw new Error(`Unsupported log drain sink: ${variables.destination satisfies never}`);
      }
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
    switch (drain.kind) {
      case "http": {
        const fields = editableHeaders(drain.config.headers, nextHeaderId.current);
        nextHeaderId.current += fields.length;
        setHeaders(fields);
        break;
      }
      case "axiom":
        break;
      default:
        throw new Error(`Unsupported log drain sink: ${drain satisfies never}`);
    }
  }, [drain]);

  const currentDestination = getDestination(drain);
  const saveDestination = () => {
    const value = destination.trim();
    switch (drain.kind) {
      case "http":
        update.mutate({ id: drain.id, destination: { kind: "http", config: { url: value } } });
        break;
      case "axiom":
        update.mutate({ id: drain.id, destination: { kind: "axiom", config: { dataset: value } } });
        break;
      default:
        throw new Error(`Unsupported log drain sink: ${drain satisfies never}`);
    }
  };
  const saveHeaders = () => {
    const requested: HeaderUpdateField[] = [];
    for (const header of headers) {
      const name = header.name.trim();
      if (header.source === "new" && name === "" && header.value === "") {
        continue;
      }
      if (header.source === "stored" && header.value === "") {
        requested.push({ mode: "preserve", name });
      } else {
        requested.push({ mode: "set", name, value: header.value });
      }
    }
    const parsed = headerUpdateFieldsSchema.safeParse(requested);
    if (!parsed.success) {
      toast.error(parsed.error.issues[0]?.message ?? "Check the header fields");
      return;
    }
    switch (drain.kind) {
      case "http":
        update.mutate({
          id: drain.id,
          destination: {
            kind: "http",
            config: { headers: parsed.data },
          },
        });
        break;
      case "axiom":
        break;
      default:
        throw new Error(`Unsupported log drain sink: ${drain satisfies never}`);
    }
  };
  const saveToken = () => {
    switch (drain.kind) {
      case "http":
        break;
      case "axiom":
        update.mutate({
          id: drain.id,
          destination: {
            kind: "axiom",
            config: { token: token.trim() },
          },
        });
        break;
      default:
        throw new Error(`Unsupported log drain sink: ${drain satisfies never}`);
    }
  };
  const headersChanged =
    headers.filter((header) => header.source === "stored").length !== configuredHeaders.length ||
    headers.some((header) =>
      header.source === "stored"
        ? header.value !== ""
        : header.name.trim() !== "" || header.value !== "",
    );
  let sinkSettings: ReactNode;
  switch (drain.kind) {
    case "http":
      sinkSettings = (
        <>
          <SettingCard
            title="Headers"
            description="Header values stay hidden after you save them. Leave a value blank to keep its current value."
            contentWidth="w-full lg:w-[520px] justify-end"
          >
            <div className="flex w-full flex-col gap-3">
              {headers.map((header, index) => (
                <div key={header.id} className="flex items-center gap-2">
                  <Input
                    aria-label={`Header ${index + 1} name`}
                    placeholder="Header name"
                    value={header.name}
                    disabled={header.source === "stored"}
                    onChange={(event) =>
                      setHeaders((current) =>
                        current.map((item, itemIndex) =>
                          itemIndex === index ? { ...item, name: event.target.value } : item,
                        ),
                      )
                    }
                  />
                  <Input
                    type="password"
                    autoComplete="off"
                    aria-label={`Header ${index + 1} value`}
                    placeholder={header.source === "stored" ? "Enter a new value" : "Header value"}
                    value={header.value}
                    onChange={(event) =>
                      setHeaders((current) =>
                        current.map((item, itemIndex) =>
                          itemIndex === index ? { ...item, value: event.target.value } : item,
                        ),
                      )
                    }
                  />
                  {header.source === "stored" || headers.length > 1 ? (
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
                    setHeaders((current) => [...current, { id, source: "new", ...emptyHeader }]);
                  }}
                >
                  <Plus iconSize="sm-regular" />
                  Add header
                </Button>
                <Button
                  variant="primary"
                  loading={update.isLoading}
                  disabled={!headersChanged}
                  onClick={saveHeaders}
                >
                  Save headers
                </Button>
              </div>
            </div>
          </SettingCard>
          <SettingCard
            title="Body format"
            description="JSON sends an array of events. NDJSON sends one event per line."
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
                          config: { format: value },
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
        </>
      );
      break;
    case "axiom":
      sinkSettings = (
        <SettingCard
          title="Token"
          description="Enter a new Axiom API token to replace the current token."
          contentWidth="w-full lg:w-[420px] justify-end"
        >
          <Input
            type="password"
            aria-label="Axiom API token"
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
            Save token
          </Button>
        </SettingCard>
      );
      break;
    default:
      throw new Error(`Unsupported log drain sink: ${drain satisfies never}`);
  }

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
            <SinkType kind={drain.kind} />
          </span>
        </PageHeaderActions>
      </PageHeader>
      <PageBody aria-live="polite">
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
              title="Name"
              description="Shown in the log drain list."
              contentWidth="w-full lg:w-[420px] justify-end"
            >
              <Input
                aria-label="Name"
                value={name}
                onChange={(event) => setName(event.target.value)}
              />
              <Button
                variant="primary"
                loading={update.isLoading}
                disabled={!name.trim() || name.trim() === drain.name}
                onClick={() => update.mutate({ id: drain.id, name: name.trim() })}
              >
                Save name
              </Button>
            </SettingCard>
            <Item className="flex-col items-stretch gap-6 py-[18px] lg:flex-row lg:items-center">
              <ItemContent>
                <ItemTitle>Log drain ID</ItemTitle>
                <ItemDescription>Unkey uses this ID to track delivery progress.</ItemDescription>
              </ItemContent>
              <ItemActions className="w-full justify-end lg:w-[420px]">
                <div className="flex min-w-0 w-full items-center justify-between rounded-lg border border-grayA-5 bg-gray-2 px-2 py-2 dark:bg-black">
                  <code className="truncate text-sm text-gray-11" translate="no">
                    {drain.id}
                  </code>
                  <CopyButton value={drain.id} variant="ghost" toastMessage={drain.id} />
                </div>
              </ItemActions>
            </Item>
            <SettingCard
              title={destinationLabels.title}
              description={destinationLabels.description}
              contentWidth="w-full lg:w-[420px] justify-end"
            >
              <Input
                aria-label={destinationLabels.title}
                value={destination}
                onChange={(event) => setDestination(event.target.value)}
              />
              <Button
                variant="primary"
                loading={update.isLoading}
                disabled={!destination.trim() || destination.trim() === currentDestination}
                onClick={saveDestination}
              >
                {destinationLabels.saveLabel}
              </Button>
            </SettingCard>
            {sinkSettings}
            <SettingCard
              title="Delivery status"
              description="Pause delivery without losing progress. Resume delivery when you are ready."
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
                  ? "Resume delivery"
                  : drain.status === "running"
                    ? "Pause delivery"
                    : "Resume delivery"}
              </Button>
            </SettingCard>
          </SettingCardGroup>
        </section>
        <SettingsDangerZone>
          <SettingsZoneRow
            title="Delete log drain"
            description="Stop all future deliveries and delete this log drain."
            action={{
              label: "Delete log drain",
              onClick: () => setConfirmDelete(true),
            }}
          />
        </SettingsDangerZone>
        <DialogContainer
          isOpen={confirmDelete}
          onOpenChange={setConfirmDelete}
          title={`Delete ${drain.name}?`}
          subTitle="Unkey will stop all future deliveries and delete this log drain."
          footer={
            <div className="flex w-full gap-2">
              <Button className="flex-1" variant="outline" onClick={() => setConfirmDelete(false)}>
                Keep log drain
              </Button>
              <Button
                className="flex-1"
                variant="primary"
                color="danger"
                loading={remove.isLoading}
                onClick={() => remove.mutate({ id: drain.id })}
              >
                Delete log drain
              </Button>
            </div>
          }
        >
          <p className="text-sm text-gray-10">You cannot undo this action.</p>
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
  const [detailsOpen, setDetailsOpen] = useState(false);

  if (loading) {
    return null;
  }

  if (error) {
    return (
      <AlertBanner variant="error">
        <AlertBannerTitle>Could not load delivery error details</AlertBannerTitle>
        <AlertBannerDescription>Refresh the page to try again.</AlertBannerDescription>
      </AlertBanner>
    );
  }

  if (!entries?.length) {
    return null;
  }

  return (
    <Collapsible open={detailsOpen} onOpenChange={setDetailsOpen} className="flex flex-col gap-2">
      <AlertBanner variant="error">
        <TriangleWarning2 iconSize="md-regular" aria-hidden="true" />
        <AlertBannerTitle>
          {entries.length === 20
            ? "At least 20 delivery attempts failed"
            : `${entries.length} delivery ${entries.length === 1 ? "attempt" : "attempts"} failed`}
        </AlertBannerTitle>
        <AlertBannerDescription>
          Review failures from the past 24 hours to find the cause. Unkey pauses the log drain when
          delivery attempts keep failing.
        </AlertBannerDescription>
        <AlertBannerActions>
          <CollapsibleTrigger
            render={<Button variant="outline" size="md" />}
            className="[&[data-panel-open]_.error-chevron]:rotate-180"
          >
            {detailsOpen ? "Hide details" : "View details"}
            <ChevronDown
              iconSize="sm-regular"
              className="error-chevron text-gray-9 transition-transform duration-200"
              aria-hidden="true"
            />
          </CollapsibleTrigger>
        </AlertBannerActions>
      </AlertBanner>
      <CollapsibleContent>
        <div className="overflow-x-auto rounded-lg border border-grayA-4 bg-background">
          <table className="w-full min-w-[680px] table-fixed border-collapse text-left">
            <colgroup>
              <col className="w-44" />
              <col className="w-32" />
              <col />
            </colgroup>
            <thead className="bg-grayA-2">
              <tr className="border-b border-grayA-4">
                <th className="px-4 py-2 text-xs font-medium text-gray-10">Time</th>
                <th className="px-4 py-2 text-xs font-medium text-gray-10">Result</th>
                <th className="px-4 py-2 text-xs font-medium text-gray-10">Details</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-grayA-4">
              {entries.map((entry) => {
                const detail =
                  entry.responseBody ||
                  entry.error ||
                  "The destination did not return error details.";
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
          <Skeleton className="h-5 w-48" />
        </PageHeaderContent>
      </PageHeader>
      <PageBody aria-busy="true" aria-live="polite">
        <output className="sr-only">Loading log drain…</output>
        <Skeleton className="h-52 rounded-xl" />
        <Skeleton className="h-96 rounded-xl" />
      </PageBody>
    </PageContainer>
  );
}
