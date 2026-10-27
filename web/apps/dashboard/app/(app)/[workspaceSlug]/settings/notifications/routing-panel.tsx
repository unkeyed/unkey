"use client";

import { useNotificationRecipients } from "@/lib/notifications/use-notifications";
import { getErrorMessage } from "@/lib/unkey-client";
import {
  Badge,
  Button,
  FormSelect,
  SlidePanel,
  SlidePanelCloseButton,
  SlidePanelContent,
  SlidePanelDescription,
  SlidePanelHeader,
  SlidePanelTitle,
} from "@unkey/ui";
import { useState } from "react";
import { useNotifications } from "./notification-context";
import { ChannelIcon, useResourceNames } from "./notification-ui";

export function RoutingPanel({
  isOpen,
  onClose,
  initialType,
}: {
  isOpen: boolean;
  onClose: () => void;
  initialType?: string;
}) {
  const { types, destinations, currentUserId } = useNotifications();
  const resources = useResourceNames();
  const [notificationType, setNotificationType] = useState(initialType ?? types[0]?.type ?? "");
  const [projectId, setProjectId] = useState("");
  const [appId, setAppId] = useState("");
  const [environmentId, setEnvironmentId] = useState("");
  const project = resources.projects.find((candidate) => candidate.id === projectId);
  const app = project?.apps.find((candidate) => candidate.id === appId);
  const environment = resources.environments.find(
    (candidate) => candidate.id === environmentId && candidate.appId === appId,
  );
  const complete = Boolean(
    types.some((type) => type.type === notificationType) && project && app && environment,
  );
  const recipients = useNotificationRecipients(
    isOpen && complete && !resources.isError
      ? { notificationType, projectId, appId, environmentId }
      : null,
  );
  const eligible = recipients.data?.filter((recipient) => !recipient.reason) ?? [];
  const emailCount = eligible.filter((recipient) => recipient.channel === "email").length;

  return (
    <SlidePanel isOpen={isOpen} onClose={onClose} widthClassName="w-160">
      <SlidePanelHeader className="items-start">
        <div className="flex flex-col gap-1">
          <SlidePanelTitle>Preview recipients</SlidePanelTitle>
          <SlidePanelDescription className="text-xs text-gray-9">
            Check saved routes for a specific environment. Nothing is sent.
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
              options={types.map((type) => ({
                value: type.type,
                label: `${type.label} (${type.type})`,
              }))}
            />
            <div className="grid grid-cols-3 gap-3">
              <FormSelect
                label="Project"
                placeholder="Choose project"
                value={projectId}
                disabled={resources.isLoading || resources.isError}
                onValueChange={(next) => {
                  setProjectId(next);
                  setAppId("");
                  setEnvironmentId("");
                }}
                options={resources.projects.map((item) => ({ value: item.id, label: item.name }))}
              />
              <FormSelect
                label="App"
                placeholder="Choose app"
                value={appId}
                disabled={!project || resources.isLoading || resources.isError}
                onValueChange={(next) => {
                  setAppId(next);
                  setEnvironmentId("");
                }}
                options={(project?.apps ?? []).map((item) => ({
                  value: item.id,
                  label: item.name,
                }))}
              />
              <FormSelect
                label="Environment"
                placeholder="Choose environment"
                value={environmentId}
                disabled={!app || resources.isLoading || resources.isError}
                onValueChange={setEnvironmentId}
                options={resources.environments
                  .filter((item) => item.appId === appId)
                  .map((item) => ({ value: item.id, label: item.slug }))}
              />
            </div>
          </div>
          {resources.isError ? (
            <p role="alert" className="text-xs text-error-11">
              Couldn't load resources. Refresh to try again.
            </p>
          ) : resources.isLoading ? (
            <output className="text-xs text-gray-11">Loading resources…</output>
          ) : complete ? (
            recipients.isError ? (
              <div className="space-y-2">
                <p role="alert" className="text-xs text-error-11">
                  {getErrorMessage(recipients.error, "Couldn't preview recipients.")}
                </p>
                <Button variant="outline" size="sm" onClick={() => recipients.refetch()}>
                  Retry
                </Button>
              </div>
            ) : recipients.isFetching ? (
              <output className="text-sm text-gray-11">Checking recipients…</output>
            ) : recipients.data ? (
              <>
                <output aria-live="polite" className="text-sm text-gray-12">
                  {eligible.length === 0
                    ? "No eligible recipients for this event."
                    : `${emailCount} email ${emailCount === 1 ? "recipient" : "recipients"} and ${eligible.length - emailCount} shared ${eligible.length - emailCount === 1 ? "destination" : "destinations"} eligible.`}
                </output>
                {(["Email", "Destinations"] as const).map((section) => {
                  const items = recipients.data.filter(
                    (recipient) => (recipient.channel === "email") === (section === "Email"),
                  );
                  return (
                    <section key={section} className="flex flex-col gap-2">
                      <h3 className="text-xs font-medium uppercase tracking-wide text-gray-9">
                        {section} ({items.length})
                      </h3>
                      {items.length === 0 ? (
                        <p className="text-xs text-gray-9">None.</p>
                      ) : (
                        <ul className="divide-y divide-grayA-4 rounded-lg border bg-raised">
                          {items.map((recipient) => {
                            const destination = destinations.find(
                              (item) => item.id === recipient.destinationId,
                            );
                            return (
                              <li key={recipient.key} className="flex items-start gap-3 px-4 py-3">
                                <ChannelIcon
                                  channel={
                                    recipient.channel === "email"
                                      ? "email"
                                      : (destination?.channel ?? "webhook")
                                  }
                                  className="mt-0.5 size-4 text-gray-11"
                                />
                                <div className="flex min-w-0 flex-1 flex-col gap-0.5">
                                  <span className="text-sm text-gray-12">
                                    {recipient.email ??
                                      destination?.name ??
                                      recipient.destinationId ??
                                      recipient.key}
                                    {recipient.userId === currentUserId ? (
                                      <span className="ml-1 text-xs text-gray-9">(you)</span>
                                    ) : null}
                                  </span>
                                  {recipient.reason ? (
                                    <span className="text-xs text-gray-10">{recipient.reason}</span>
                                  ) : null}
                                </div>
                                <Badge variant={recipient.reason ? "secondary" : "primary"}>
                                  {recipient.reason ? "Ineligible" : "Eligible"}
                                </Badge>
                              </li>
                            );
                          })}
                        </ul>
                      )}
                    </section>
                  );
                })}
              </>
            ) : null
          ) : (
            <p className="text-sm text-gray-11">
              Choose a project, app, and environment to preview recipients.
            </p>
          )}
        </div>
      </SlidePanelContent>
    </SlidePanel>
  );
}
