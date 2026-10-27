"use client";

import { Switch } from "@/components/ui/switch";
import { notificationError } from "@/lib/notifications/use-notifications";
import { IconPlusOutline12 } from "@unkey/icons";
import {
  Badge,
  Button,
  EmptyState,
  EmptyStateActions,
  EmptyStateDescription,
  EmptyStateHeader,
  EmptyStateTitle,
  ResourceListBody,
  ResourceListContent,
  ResourceListHeader,
  ResourceListItem,
  SettingsGroup,
  SettingsGroupTitle,
  toast,
} from "@unkey/ui";
import { useState } from "react";
import type { DestinationEditorTarget } from "./destination-panel";
import { useNotifications } from "./notification-context";
import { rulesUsingDestination } from "./notification-model";
import { ChannelMedia, channelTitle } from "./notification-ui";

export function ConnectedChannels({
  onEditDestination,
}: {
  onEditDestination: (target: DestinationEditorTarget) => void;
}) {
  const notifications = useNotifications();
  const [pendingId, setPendingId] = useState<string | null>(null);
  const addButton = (
    <Button
      type="button"
      variant="outline"
      size="sm"
      disabled={!notifications.canManage}
      onClick={() => onEditDestination({ mode: "create" })}
    >
      <IconPlusOutline12 />
      Connect channel
    </Button>
  );

  return (
    <SettingsGroup>
      <ResourceListHeader className="justify-between gap-3">
        <div className="flex flex-col gap-1">
          <SettingsGroupTitle>Connected channels</SettingsGroupTitle>
          <p className="text-xs leading-5 text-gray-11">
            Reuse a channel across events. Pausing it stops every route that uses it.
          </p>
        </div>
        <div className="shrink-0">{addButton}</div>
      </ResourceListHeader>

      {notifications.destinations.length === 0 ? (
        <EmptyState>
          <EmptyStateHeader>
            <EmptyStateTitle>No connected channels</EmptyStateTitle>
            <EmptyStateDescription>
              Rules can still email members. Connect Slack, Discord, or your own endpoint.
            </EmptyStateDescription>
          </EmptyStateHeader>
          <EmptyStateActions>{addButton}</EmptyStateActions>
        </EmptyState>
      ) : (
        <ResourceListContent>
          <ResourceListBody>
            {notifications.destinations.map((destination) => {
              const usedBy = rulesUsingDestination(notifications.rules, destination.id).length;
              return (
                <ResourceListItem
                  key={destination.id}
                  className="flex items-center gap-3 px-4 py-4 transition-colors hover:bg-grayA-2"
                >
                  <button
                    type="button"
                    disabled={!notifications.canManage || pendingId !== null}
                    onClick={() => onEditDestination({ mode: "edit", destination })}
                    aria-label={`Edit ${destination.name}`}
                    className="group flex min-w-0 flex-1 items-center gap-3 rounded-md text-left focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-grayA-7 enabled:cursor-pointer"
                  >
                    <ChannelMedia channel={destination.channel} />
                    <span className="flex min-w-0 flex-1 flex-col">
                      <span className="flex items-center gap-2">
                        <span className="truncate group-hover:underline text-sm font-medium text-gray-12">
                          {destination.name}
                        </span>
                        {destination.enabled ? null : <Badge variant="secondary">Paused</Badge>}
                      </span>
                      <span className="truncate text-xs text-gray-9">
                        {channelTitle(destination.channel)} ·{" "}
                        <span className="font-mono">{destination.urlHint}</span>
                      </span>
                    </span>
                    <span className="hidden shrink-0 text-xs text-gray-11 sm:block">
                      {usedBy === 0
                        ? "Not used by any rule"
                        : `Used by ${usedBy} ${usedBy === 1 ? "rule" : "rules"}`}
                    </span>
                  </button>
                  <Switch
                    size="sm"
                    aria-label={`Deliver to ${destination.name}`}
                    disabled={!notifications.canManage || pendingId !== null}
                    aria-busy={pendingId === destination.id}
                    checked={destination.enabled}
                    onCheckedChange={async (enabled) => {
                      setPendingId(destination.id);
                      try {
                        await notifications.actions.updateDestination({
                          destinationId: destination.id,
                          enabled,
                        });
                        toast.success(
                          enabled ? `${destination.name} resumed` : `${destination.name} paused`,
                        );
                      } catch (error) {
                        notificationError(error, "Couldn't update destination");
                      } finally {
                        setPendingId(null);
                      }
                    }}
                  />
                </ResourceListItem>
              );
            })}
          </ResourceListBody>
        </ResourceListContent>
      )}
    </SettingsGroup>
  );
}
