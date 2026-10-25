"use client";

import { Switch } from "@/components/ui/switch";
import { IconPlusOutline12 } from "@unkey/icons";
import {
  Badge,
  Button,
  EmptyState,
  EmptyStateActions,
  EmptyStateDescription,
  EmptyStateHeader,
  EmptyStateTitle,
  ResourceList,
  ResourceListBody,
  ResourceListContent,
  ResourceListHeader,
  ResourceListItem,
  toast,
} from "@unkey/ui";
import type { DestinationEditorTarget } from "./destination-panel";
import { rulesUsingDestination } from "./notification-model";
import { ChannelMedia, channelTitle } from "./notification-ui";
import { useNotificationPreview } from "./preview-context";

export function DestinationsTab({
  onEditDestination,
}: {
  onEditDestination: (target: DestinationEditorTarget) => void;
}) {
  const preview = useNotificationPreview();
  const addButton = (
    <Button
      variant="primary"
      disabled={!preview.canManage}
      onClick={() => onEditDestination({ mode: "create" })}
    >
      <IconPlusOutline12 />
      Add destination
    </Button>
  );

  return (
    <ResourceList className="gap-6">
      <ResourceListHeader className="justify-between gap-4">
        <p className="max-w-xl text-sm text-gray-11">
          Set up a Slack channel, Discord channel, or webhook once, then add it to any rule. Pausing
          a destination stops every rule from posting to it.
        </p>
        <div className="shrink-0">{addButton}</div>
      </ResourceListHeader>

      {preview.destinations.length === 0 ? (
        <EmptyState>
          <EmptyStateHeader>
            <EmptyStateTitle>No destinations</EmptyStateTitle>
            <EmptyStateDescription>
              Rules can still email members. Add a destination to post to Slack, Discord, or your
              own endpoint.
            </EmptyStateDescription>
          </EmptyStateHeader>
          <EmptyStateActions>{addButton}</EmptyStateActions>
        </EmptyState>
      ) : (
        <ResourceListContent>
          <ResourceListBody>
            {preview.destinations.map((destination) => {
              const usedBy = rulesUsingDestination(preview.rules, destination.id).length;
              return (
                <ResourceListItem
                  key={destination.id}
                  className="flex items-center gap-3 px-4 py-4 transition-colors hover:bg-grayA-2"
                >
                  <button
                    type="button"
                    disabled={!preview.canManage}
                    onClick={() => onEditDestination({ mode: "edit", destination })}
                    aria-label={`Edit ${destination.name}`}
                    className="group flex min-w-0 flex-1 items-center gap-3 rounded-md text-left focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-grayA-7 enabled:cursor-pointer"
                  >
                    <ChannelMedia channel={destination.config.channel} />
                    <span className="flex min-w-0 flex-1 flex-col">
                      <span className="flex items-center gap-2">
                        <span className="truncate group-hover:underline text-sm font-medium text-gray-12">
                          {destination.name}
                        </span>
                        {destination.enabled ? (
                          <Badge variant="success">Active</Badge>
                        ) : (
                          <Badge variant="secondary">Paused</Badge>
                        )}
                      </span>
                      <span className="truncate text-xs text-gray-9">
                        {channelTitle(destination.config.channel)} ·{" "}
                        <span className="font-mono">{destination.config.urlHint}</span>
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
                    disabled={!preview.canManage}
                    checked={destination.enabled}
                    onCheckedChange={(enabled) => {
                      preview.actions.setDestinationEnabled(destination.id, enabled);
                      toast.success(
                        enabled
                          ? `${destination.name} resumed in preview`
                          : `${destination.name} paused in preview`,
                      );
                    }}
                  />
                </ResourceListItem>
              );
            })}
          </ResourceListBody>
        </ResourceListContent>
      )}
      <p className="text-xs text-gray-11">
        Active means enabled, not verified. Open a destination to view its configuration.
      </p>
    </ResourceList>
  );
}
