"use client";

import { type MenuItem, TableActionPopover } from "@/components/logs/table-action.popover";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { routes } from "@/lib/navigation/routes";
import { Bolt, ChevronLeft, Dots, Gear, MediaPause, Trash } from "@unkey/icons";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  Button,
  PageBody,
  PageContainer,
  PageHeader,
  PageHeaderActions,
  PageHeaderContent,
  PageHeaderTitle,
} from "@unkey/ui";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { useState } from "react";
import { DrainErrorBanner } from "../drain-error-banner";
import type { DrainDetail } from "../drain-schema";
import { DrainStatusBadge } from "../drain-status-badge";
import { DrainLogsTable } from "./drain-logs-table";
import { DrainSettingsPanel } from "./drain-settings-panel";
import { DrainStatsCards } from "./drain-stats-cards";
import { useDrainSettings } from "./use-drain-settings";

/**
 * The page is read-only. Everything that writes lives behind the Settings panel or the overflow
 * menu, so the page itself never holds an unsaved edit.
 */
export function LogdrainDetail({ drain }: { drain: DrainDetail }) {
  const workspace = useWorkspaceNavigation();
  const router = useRouter();
  const listHref = routes.settings.logdrains.list({ workspaceSlug: workspace.slug });

  const settings = useDrainSettings(drain, { onDeleted: () => router.push(listHref) });
  const { toggleStatus, setStatus, remove, confirmDelete, setConfirmDelete } = settings;
  const [isSettingsOpen, setIsSettingsOpen] = useState(false);
  const running = drain.status === "running";

  const menuItems: MenuItem[] = [
    {
      id: "pause",
      label: running ? "Pause deliveries" : "Resume deliveries",
      icon: running ? <MediaPause iconSize="md-regular" /> : <Bolt iconSize="md-regular" />,
      disabled: setStatus.isLoading,
      onClick: () => toggleStatus(),
    },
    {
      id: "delete",
      label: "Delete log drain",
      icon: <Trash iconSize="md-regular" />,
      onClick: () => setConfirmDelete(true),
    },
  ];

  return (
    <PageContainer>
      <PageHeader>
        <PageHeaderContent className="flex-1">
          <Link
            href={listHref}
            className="-ml-1 flex w-fit items-center gap-1 rounded-md px-1 py-0.5 text-[13px] text-gray-10 transition-colors hover:text-gray-12 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-grayA-7"
          >
            <ChevronLeft iconSize="sm-regular" />
            Log Drains
          </Link>
          <div className="flex min-w-0 items-center gap-3">
            <PageHeaderTitle className="truncate">{drain.name}</PageHeaderTitle>
            <DrainStatusBadge status={drain.status} />
          </div>
        </PageHeaderContent>
        <PageHeaderActions>
          <Button variant="outline" onClick={() => setIsSettingsOpen(true)}>
            <Gear iconSize="sm-regular" />
            Settings
          </Button>
          <TableActionPopover items={menuItems}>
            <Button variant="outline" className="w-7 p-0" aria-label="Open actions">
              <Dots iconSize="sm-regular" />
            </Button>
          </TableActionPopover>
        </PageHeaderActions>
      </PageHeader>

      <PageBody>
        <DrainErrorBanner status={drain.status} />
        <DrainStatsCards drainId={drain.id} />
        <DrainLogsTable drainId={drain.id} />
      </PageBody>

      <DrainSettingsPanel
        settings={settings}
        isOpen={isSettingsOpen}
        onClose={() => setIsSettingsOpen(false)}
      />

      <AlertDialog open={confirmDelete} onOpenChange={setConfirmDelete}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Delete {drain.name}?</AlertDialogTitle>
            <AlertDialogDescription>
              Unkey will stop all future deliveries and delete this log drain. You cannot undo this
              action.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Keep log drain</AlertDialogCancel>
            <AlertDialogAction color="danger" onClick={() => remove.mutate({ id: drain.id })}>
              Delete log drain
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </PageContainer>
  );
}
