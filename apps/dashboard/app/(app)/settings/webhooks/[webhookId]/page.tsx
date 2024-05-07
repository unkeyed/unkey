import { BackLink } from "@/components/back";
import { CopyButton } from "@/components/dashboard/copy-button";
import { PageHeader } from "@/components/dashboard/page-header";
import { Badge } from "@/components/ui/badge";
import { Code } from "@/components/ui/code";
import { Separator } from "@/components/ui/separator";
import { type Event, db } from "@/lib/db";
import { cn } from "@/lib/utils";
import { Check, CheckCircle } from "lucide-react";
import ms from "ms";
import { Solitreo } from "next/font/google";
import Link from "next/link";
import { notFound, redirect } from "next/navigation";
import { DeleteWebhook } from "./delete-webhook";
import { Until } from "./duration";
import { ToggleWebhookButton } from "./toggle-webhook-button";
type Props = {
  params: {
    webhookId: string;
  };
  searchParams: {
    eventId?: string;
  };
};

export default async function Page(props: Props) {
  const webhook = await db.query.webhooks.findFirst({
    where: (table, { eq }) => eq(table.id, props.params.webhookId),
    columns: {
      id: true,
      enabled: true,
      destination: true,
    },
    with: {
      events: {
        limit: 50,
        orderBy: (table, { desc }) => desc(table.time),
        with: {
          deliveryAttempts: {
            orderBy: (table, { desc }) => desc(table.time),
          },
        },
      },
    },
  });
  if (!webhook) {
    return notFound();
  }

  const selectedEvent =
    webhook.events.find((event) => event.id === props.searchParams.eventId) ?? webhook.events.at(0);
  return (
    <div>
      <BackLink href="/settings/webhooks" label="Back" />
      <PageHeader
        title={webhook.destination}
        actions={[
          selectedEvent ? (
            <Badge
              key="selectedEvent"
              variant="secondary"
              className="flex justify-between w-full gap-2 font-mono font-medium ph-no-capture"
            >
              {selectedEvent.id}
              <CopyButton value={selectedEvent.id} />
            </Badge>
          ) : null,
          <ToggleWebhookButton
            key="toggle"
            webhook={{ id: webhook.id, enabled: webhook.enabled }}
          />,
          <DeleteWebhook
            key="delete"
            webhook={{ id: webhook.id, destination: webhook.destination }}
          />,
        ]}
      />

      <div className="flex gap-8">
        <ul className="flex flex-col w-1/3 overflow-hidden border divide-y rounded-lg divide-border bg-background border-border">
          {webhook.events.map((event) => (
            <Link
              href={`/settings/webhooks/${props.params.webhookId}?eventId=${event.id}`}
              key={event.id}
              className={cn(
                "flex items-center justify-between w-full px-4 py-2 duration-250 hover:bg-background-subtle ",
                {
                  "bg-background-subtle": event.id === selectedEvent?.id,
                },
              )}
            >
              <div className="flex flex-col items-start col-span-8 ">
                <span className="font-mono text-sm text-content">{event.event}</span>
                <time
                  dateTime={new Date(event.time).toISOString()}
                  className="text-xs whitespace-nowrap text-content-subtle"
                >
                  {new Date(event.time).toISOString()}
                </time>
              </div>

              <StateIndicator state={event.state} />
            </Link>
          ))}
        </ul>

        <div className="w-2/3">
          {selectedEvent ? (
            <div className="flex flex-col gap-8 ">
              <Separator />
              <div className="">
                <ol className="flex flex-col">
                  {selectedEvent.deliveryAttempts.map((attempt, i) => (
                    <li className="relative flex pb-12 gap-x-4" key={attempt.id}>
                      <div
                        className={cn(
                          i === selectedEvent.deliveryAttempts.length - 1 ? "h-6" : "-bottom-6",
                          "absolute left-0 top-0 flex w-6 justify-center",
                        )}
                      >
                        <div className="w-px bg-border" />
                      </div>
                      <div className="relative flex items-center justify-center flex-none w-6 h-6 bg-background">
                        <div
                          className={cn("h-1.5 w-1.5 rounded-full", {
                            "bg-alert": !attempt.success,
                            "bg-success": attempt.success,
                          })}
                        />
                      </div>

                      <div className="flex flex-col w-full gap-4 ">
                        <time className="inline-flex items-center h-6 font-mono text-sm leading-none text-content">
                          {new Date(attempt.time).toString()}
                        </time>
                        <dl className="grid grid-cols-2 gap-2">
                          {attempt.internalError ? (
                            <>
                              <dt className="font-mono text-xs text-content-subtle">Error</dt>
                              <dd className="font-mono text-sm text-alert">
                                {attempt.internalError}
                              </dd>
                            </>
                          ) : null}
                          {attempt.responseStatus ? (
                            <>
                              <dt className="font-mono text-xs text-content-subtle">Status</dt>
                              <dd className="font-mono text-sm text-content">
                                {attempt.responseStatus}
                              </dd>
                            </>
                          ) : null}
                          {attempt.nextAttemptAt && attempt.nextAttemptAt > Date.now() ? (
                            <>
                              <dt className="font-mono text-xs text-content-subtle">Retrying in</dt>
                              <dd className="font-mono text-sm text-content">
                                <Until time={attempt.nextAttemptAt} />
                              </dd>
                            </>
                          ) : null}
                        </dl>
                        {attempt.responseBody ? (
                          <Code>{JSON.stringify(attempt.responseBody)}</Code>
                        ) : null}
                      </div>
                    </li>
                  ))}
                </ol>
              </div>

              <Separator />
              <div className="">
                <h3 className="font-mono text-sm font-medium">Request</h3>

                <Code>{JSON.stringify(selectedEvent.payload, null, 2)}</Code>
              </div>
            </div>
          ) : null}
        </div>
      </div>
    </div>
  );
}

const StateIndicator: React.FC<{ state: Event["state"] }> = ({ state }) => {
  return (
    <span className="inline-flex items-center gap-x-1.5 rounded-md px-2 py-1 text-xs font-medium text-gray-900 ring-1 ring-inset ring-gray-200">
      <svg
        className={cn("h-1.5 w-1.5", {
          "fill-alert": state === "failed",
          "fill-success": state === "delivered",
          "fill-gray-500": state === "retrying" || state === "created",
        })}
        viewBox="0 0 6 6"
        aria-hidden="true"
      >
        <circle cx={3} cy={3} r={3} />
      </svg>
      {state}
    </span>
  );
};
