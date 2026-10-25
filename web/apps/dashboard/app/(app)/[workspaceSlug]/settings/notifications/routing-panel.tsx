"use client";

import {
  Badge,
  FormSelect,
  SlidePanel,
  SlidePanelCloseButton,
  SlidePanelContent,
  SlidePanelDescription,
  SlidePanelHeader,
  SlidePanelTitle,
} from "@unkey/ui";
import { useState } from "react";
import { type RouteReason, resolveRecipients, wasDeduplicated } from "./notification-model";
import { NOTIFICATION_TYPES, findNotificationType } from "./notification-types";
import { ChannelIcon, useResourceNames } from "./notification-ui";
import { useNotificationPreview } from "./preview-context";
import { isSampleMember } from "./preview-store";

const ANY = "any";

function Reasons({ reasons }: { reasons: ReadonlyArray<RouteReason> }) {
  return (
    <span className="text-xs text-gray-9">
      {reasons.map((reason) => `${reason.ruleName} (${reason.via.join(", ")})`).join(" · ")}
    </span>
  );
}

export function RoutingPanel({
  isOpen,
  onClose,
  initialType,
}: {
  isOpen: boolean;
  onClose: () => void;
  initialType?: string;
}) {
  const preview = useNotificationPreview();
  const resources = useResourceNames();
  const [notificationType, setNotificationType] = useState(
    initialType ?? NOTIFICATION_TYPES[0].type,
  );
  const [projectId, setProjectId] = useState(ANY);
  const [appId, setAppId] = useState(ANY);
  const [environmentId, setEnvironmentId] = useState(ANY);

  const definition = findNotificationType(notificationType);
  const project = resources.projects.find((candidate) => candidate.id === projectId);
  const scoped = definition?.resourceScoped ?? false;
  const resolved = resolveRecipients(preview, {
    notificationType,
    resource: scoped
      ? {
          projectId: projectId === ANY ? undefined : projectId,
          appId: appId === ANY ? undefined : appId,
          environmentId: environmentId === ANY ? undefined : environmentId,
        }
      : {},
  });
  const matchedRules = new Set(
    [...resolved.email, ...resolved.destinations].flatMap((recipient) =>
      recipient.reasons.map((reason) => reason.ruleId),
    ),
  ).size;
  const sentEmails = resolved.email.filter((recipient) => recipient.status === "send").length;
  const sentPosts = resolved.destinations.filter((recipient) => recipient.status === "send").length;

  return (
    <SlidePanel isOpen={isOpen} onClose={onClose} widthClassName="w-160">
      <SlidePanelHeader className="items-start">
        <div className="flex flex-col gap-1">
          <SlidePanelTitle>Test routing</SlidePanelTitle>
          <SlidePanelDescription className="text-xs text-gray-9">
            See who would hear about an event. Nothing is sent.
          </SlidePanelDescription>
        </div>
        <SlidePanelCloseButton />
      </SlidePanelHeader>
      <SlidePanelContent>
        <div className="flex h-full flex-col gap-5 overflow-y-auto px-6 pb-6 pt-1">
          <div className="flex flex-col gap-3 rounded-lg border bg-raised p-4">
            <FormSelect
              label="Event"
              value={notificationType}
              onValueChange={setNotificationType}
              options={NOTIFICATION_TYPES.map((type) => ({
                value: type.type,
                label: `${type.label} (${type.type})`,
              }))}
            />
            {scoped ? (
              <div className="grid grid-cols-3 gap-3">
                <FormSelect
                  label="Project"
                  value={projectId}
                  onValueChange={(next) => {
                    setProjectId(next);
                    setAppId(ANY);
                    setEnvironmentId(ANY);
                  }}
                  options={[
                    { value: ANY, label: "Any project" },
                    ...resources.projects.map((p) => ({ value: p.id, label: p.name })),
                  ]}
                />
                <FormSelect
                  label="App"
                  value={appId}
                  disabled={!project}
                  onValueChange={(next) => {
                    setAppId(next);
                    setEnvironmentId(ANY);
                  }}
                  options={[
                    { value: ANY, label: "Any app" },
                    ...(project?.apps ?? []).map((app) => ({ value: app.id, label: app.name })),
                  ]}
                />
                <FormSelect
                  label="Environment"
                  value={environmentId}
                  disabled={appId === ANY}
                  onValueChange={setEnvironmentId}
                  options={[
                    { value: ANY, label: "Any environment" },
                    ...resources.environments
                      .filter((env) => env.appId === appId)
                      .map((env) => ({ value: env.id, label: env.slug })),
                  ]}
                />
              </div>
            ) : null}
          </div>

          <output aria-live="polite" className="text-sm text-gray-12">
            {matchedRules === 0
              ? "No rule matches. Nobody would be notified."
              : `${matchedRules} ${matchedRules === 1 ? "rule matches" : "rules match"}. ${sentEmails} ${sentEmails === 1 ? "person gets" : "people get"} one email each, ${sentPosts} ${sentPosts === 1 ? "destination gets" : "destinations get"} one post each.`}
          </output>

          <section className="flex flex-col gap-2">
            <h3 className="text-xs font-medium uppercase tracking-wide text-gray-9">
              Email ({resolved.email.length})
            </h3>
            {resolved.email.length === 0 ? (
              <p className="text-xs text-gray-9">No email recipients.</p>
            ) : (
              <ul className="divide-y divide-grayA-4 rounded-lg border bg-raised">
                {resolved.email.map((recipient) => (
                  <li key={recipient.member.userId} className="flex items-start gap-3 px-4 py-3">
                    <ChannelIcon channel="email" className="mt-0.5 size-4 text-gray-11" />
                    <div className="flex min-w-0 flex-1 flex-col gap-0.5">
                      <span className="flex items-center gap-2 text-sm text-gray-12">
                        {recipient.member.name}
                        {recipient.member.userId === preview.currentUserId ? (
                          <span className="text-xs text-gray-9">(you)</span>
                        ) : null}
                        {isSampleMember(recipient.member.userId) ? (
                          <Badge size="sm">Sample</Badge>
                        ) : null}
                      </span>
                      <span className="text-xs text-gray-10">{recipient.member.email}</span>
                      <Reasons reasons={recipient.reasons} />
                    </div>
                    {recipient.status === "opted_out" ? (
                      <Badge variant="secondary">Turned off email</Badge>
                    ) : wasDeduplicated(recipient.reasons) ? (
                      <Badge variant="primary">Deduplicated</Badge>
                    ) : null}
                  </li>
                ))}
              </ul>
            )}
          </section>

          <section className="flex flex-col gap-2">
            <h3 className="text-xs font-medium uppercase tracking-wide text-gray-9">
              Destinations ({resolved.destinations.length})
            </h3>
            {resolved.destinations.length === 0 ? (
              <p className="text-xs text-gray-9">No destinations.</p>
            ) : (
              <ul className="divide-y divide-grayA-4 rounded-lg border bg-raised">
                {resolved.destinations.map((recipient) => (
                  <li key={recipient.destination.id} className="flex items-start gap-3 px-4 py-3">
                    <ChannelIcon
                      channel={recipient.destination.config.channel}
                      className="mt-0.5 size-4 text-gray-11"
                    />
                    <div className="flex min-w-0 flex-1 flex-col gap-0.5">
                      <span className="text-sm text-gray-12">{recipient.destination.name}</span>
                      <Reasons reasons={recipient.reasons} />
                    </div>
                    {recipient.status === "paused" ? (
                      <Badge variant="secondary">Paused</Badge>
                    ) : wasDeduplicated(recipient.reasons) ? (
                      <Badge variant="primary">Deduplicated</Badge>
                    ) : null}
                  </li>
                ))}
              </ul>
            )}
          </section>
        </div>
      </SlidePanelContent>
    </SlidePanel>
  );
}
