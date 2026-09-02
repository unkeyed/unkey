"use client";

import { match } from "@unkey/match";
import {
  Button,
  CopyButton,
  DialogContainer,
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
  toast,
} from "@unkey/ui";
import { useRouter } from "next/navigation";
import { type ReactNode, useEffect, useState } from "react";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { routes } from "@/lib/navigation/routes";
import { trpc } from "@/lib/trpc/client";
import { SinkType, StatusBadge } from "../logdrain-ui";
import { DeliveryOverview } from "./charts";
import { RecentDeliveryErrors } from "./recent-delivery-errors";
import type { Drain, DrainTelemetry } from "./types";
import type { useDrainUpdate } from "./use-drain-update";

type DrainShellProps = DrainTelemetry & {
  drain: Drain;
  destination: string;
  currentDestination: string;
  onDestinationChange: (value: string) => void;
  onSaveDestination: (value: string) => void;
  update: ReturnType<typeof useDrainUpdate>;
  children: ReactNode;
};

export function DrainShell({
  drain,
  destination,
  currentDestination,
  onDestinationChange,
  onSaveDestination,
  update,
  children,
  metricsSeries,
  metricsLoading,
  metricsError,
  recentErrorEntries,
  recentErrorsLoading,
  recentErrorsError,
}: DrainShellProps) {
  const [name, setName] = useState(drain.name);
  const [confirmDelete, setConfirmDelete] = useState(false);
  const destinationLabels = destinationCopy(drain.kind);
  const router = useRouter();
  const workspace = useWorkspaceNavigation();
  const utils = trpc.useUtils();
  const remove = trpc.logdrain.delete.useMutation({
    onSuccess: () => {
      utils.logdrain.list.invalidate();
      toast.success("Log drain deleted");
      router.push(routes.settings.logdrains.list({ workspaceSlug: workspace.slug }));
    },
    onError: (error) => toast.error(error.message),
  });

  useEffect(() => setName(drain.name), [drain.name]);

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
                onChange={(event) => onDestinationChange(event.target.value)}
              />
              <Button
                variant="primary"
                loading={update.isLoading}
                disabled={!destination.trim() || destination.trim() === currentDestination}
                onClick={() => onSaveDestination(destination.trim())}
              >
                {destinationLabels.saveLabel}
              </Button>
            </SettingCard>
            {children}
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

function destinationCopy(kind: Drain["kind"]) {
  return match(kind)
    .with("http", () => ({
      title: "HTTPS endpoint",
      description: "Unkey sends each audit log batch to this URL.",
      saveLabel: "Save endpoint",
    }))
    .with("axiom", () => ({
      title: "Dataset",
      description: "Unkey sends audit logs to this Axiom dataset.",
      saveLabel: "Save dataset",
    }))
    .exhaustive();
}
