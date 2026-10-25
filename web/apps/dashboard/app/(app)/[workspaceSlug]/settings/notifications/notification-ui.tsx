"use client";

import { useProjectEnvironments } from "@/hooks/use-project-environments";
import { useProjectsWithApps } from "@/hooks/use-projects-with-apps";
import {
  Discord,
  IconEnvelopeOutline18,
  IconLink4Outline18,
  IconUserOutline18,
} from "@unkey/icons";
import { Badge, ItemMedia, cn } from "@unkey/ui";
import { useCallback } from "react";
import {
  type Destination,
  type DestinationChannel,
  type Member,
  type RecipientTarget,
  type ResourceScope,
  audienceLabel,
} from "./notification-model";

function SlackLogo({ className }: { className?: string }) {
  return (
    <svg aria-hidden="true" className={className} viewBox="0 0 24 24" fill="currentColor">
      <path d="M5.042 15.165a2.528 2.528 0 0 1-2.52 2.523A2.528 2.528 0 0 1 0 15.165a2.527 2.527 0 0 1 2.522-2.52h2.52v2.52zm1.271 0a2.527 2.527 0 0 1 2.521-2.52 2.527 2.527 0 0 1 2.521 2.52v6.313A2.528 2.528 0 0 1 8.834 24a2.528 2.528 0 0 1-2.521-2.522v-6.313zM8.834 5.042a2.528 2.528 0 0 1-2.521-2.52A2.528 2.528 0 0 1 8.834 0a2.528 2.528 0 0 1 2.521 2.522v2.52H8.834zm0 1.271a2.528 2.528 0 0 1 2.521 2.521 2.528 2.528 0 0 1-2.521 2.521H2.522A2.528 2.528 0 0 1 0 8.834a2.528 2.528 0 0 1 2.522-2.521h6.312zm10.122 2.521a2.528 2.528 0 0 1 2.522-2.521A2.528 2.528 0 0 1 24 8.834a2.528 2.528 0 0 1-2.522 2.521h-2.522V8.834zm-1.268 0a2.528 2.528 0 0 1-2.523 2.521 2.527 2.527 0 0 1-2.52-2.521V2.522A2.527 2.527 0 0 1 15.165 0a2.528 2.528 0 0 1 2.523 2.522v6.312zm-2.523 10.122a2.528 2.528 0 0 1 2.523 2.522A2.528 2.528 0 0 1 15.165 24a2.527 2.527 0 0 1-2.52-2.522v-2.522h2.52zm0-1.268a2.527 2.527 0 0 1-2.52-2.523 2.526 2.526 0 0 1 2.52-2.52h6.313A2.527 2.527 0 0 1 24 15.165a2.528 2.528 0 0 1-2.522 2.523h-6.313z" />
    </svg>
  );
}

export const DESTINATION_CHANNELS: ReadonlyArray<{
  channel: DestinationChannel;
  title: string;
  description: string;
}> = [
  { channel: "slack", title: "Slack", description: "Post to a channel with an incoming webhook" },
  { channel: "discord", title: "Discord", description: "Post to a channel with a Discord webhook" },
  {
    channel: "webhook",
    title: "Webhook",
    description: "POST signed JSON to your own HTTPS endpoint",
  },
];

export function channelTitle(channel: DestinationChannel): string {
  return DESTINATION_CHANNELS.find((option) => option.channel === channel)?.title ?? channel;
}

export function ChannelIcon({
  channel,
  className = "size-[18px]",
}: {
  channel: DestinationChannel | "email" | "member";
  className?: string;
}) {
  if (channel === "slack") {
    return <SlackLogo className={cn(className, "p-px")} />;
  }
  if (channel === "discord") {
    return <Discord className={className} />;
  }
  if (channel === "webhook") {
    return <IconLink4Outline18 className={className} />;
  }
  if (channel === "member") {
    return <IconUserOutline18 className={className} />;
  }
  return <IconEnvelopeOutline18 className={className} />;
}

export function ChannelMedia({ channel }: { channel: DestinationChannel | "email" }) {
  return (
    <ItemMedia className="size-8 rounded-xl text-gray-12 border">
      <ChannelIcon channel={channel} />
    </ItemMedia>
  );
}

export function PreviewBadge() {
  return (
    <Badge variant="warning" size="sm">
      Preview
    </Badge>
  );
}

export function PreviewNotice() {
  return (
    <p className="text-xs leading-5 text-gray-11">
      <span className="font-medium text-gray-12">Preview</span> · Sample data. Changes stay in this
      tab; nothing is sent.
    </p>
  );
}

function TargetChip({
  icon,
  label,
  muted = false,
}: {
  icon: DestinationChannel | "email" | "member";
  label: string;
  muted?: boolean;
}) {
  return (
    <span
      className={cn(
        "inline-flex max-w-48 items-center gap-1.5 rounded-md border bg-grayA-2 px-1.5 py-0.5 text-xs text-gray-12",
        muted && "text-gray-9 line-through decoration-gray-8",
      )}
    >
      <ChannelIcon channel={icon} className="size-3 shrink-0" />
      <span className="truncate">{label}</span>
    </span>
  );
}

export function TargetChips({
  targets,
  destinations,
  members,
}: {
  targets: ReadonlyArray<RecipientTarget>;
  destinations: ReadonlyArray<Destination>;
  members: ReadonlyArray<Member>;
}) {
  if (targets.length === 0) {
    return <span className="text-xs text-warning-11">No recipients</span>;
  }
  return (
    <span className="flex flex-wrap items-center gap-1">
      {targets.map((target) => {
        if (target.kind === "audience") {
          return (
            <TargetChip
              key={`audience-${target.audience}`}
              icon="email"
              label={audienceLabel(target.audience)}
            />
          );
        }
        if (target.kind === "member") {
          const member = members.find((candidate) => candidate.userId === target.userId);
          return (
            <TargetChip
              key={`member-${target.userId}`}
              icon="member"
              label={member?.name ?? "Former member"}
              muted={!member}
            />
          );
        }
        const destination = destinations.find((candidate) => candidate.id === target.destinationId);
        return destination ? (
          <TargetChip
            key={`destination-${target.destinationId}`}
            icon={destination.config.channel}
            label={destination.enabled ? destination.name : `${destination.name} (paused)`}
            muted={!destination.enabled}
          />
        ) : null;
      })}
    </span>
  );
}

export function useResourceNames() {
  const projects = useProjectsWithApps();
  const environments = useProjectEnvironments(projects.data);

  const scopeLabel = useCallback(
    (scope: ResourceScope): string => {
      if (scope.projectId === null) {
        return "All projects";
      }
      const project = projects.data?.find((candidate) => candidate.id === scope.projectId);
      const parts = [project?.name ?? "Unknown project"];
      if (scope.appId !== null) {
        parts.push(project?.apps.find((app) => app.id === scope.appId)?.name ?? "Unknown app");
      }
      if (scope.environmentId !== null) {
        parts.push(
          environments.data?.find((environment) => environment.id === scope.environmentId)?.slug ??
            "Unknown environment",
        );
      }
      return parts.join(" / ");
    },
    [projects.data, environments.data],
  );

  return {
    projects: projects.data ?? [],
    environments: environments.data ?? [],
    isLoading: projects.isLoading,
    scopeLabel,
  };
}
