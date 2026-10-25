"use client";

import { Switch } from "@/components/ui/switch";
import { standardSchemaResolver } from "@hookform/resolvers/standard-schema";
import { IconPlusOutline12, IconTrashOutline12 } from "@unkey/icons";
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
  CopyButton,
  FormInput,
  Item,
  ItemContent,
  ItemDescription,
  ItemTitle,
  SlidePanel,
  SlidePanelCloseButton,
  SlidePanelContent,
  SlidePanelHeader,
  SlidePanelTitle,
  cn,
  toast,
} from "@unkey/ui";
import { useState } from "react";
import { useFieldArray, useForm } from "react-hook-form";
import { z } from "zod";
import {
  URL_PLACEHOLDER,
  destinationUrlProblem,
  generateSigningSecret,
  headerNameProblem,
  secretHint,
  urlHint,
} from "./destination-config";
import {
  type Destination,
  type DestinationChannel,
  type DestinationConfigSummary,
  type Rule,
  rulesUsingDestination,
} from "./notification-model";
import { ChannelMedia, DESTINATION_CHANNELS, channelTitle } from "./notification-ui";
import { useNotificationPreview } from "./preview-context";
import { newPreviewId } from "./preview-store";

export type DestinationEditorTarget =
  | { mode: "create" }
  | { mode: "edit"; destination: Destination };

function formSchema(target: DestinationEditorTarget) {
  const existingHeaders = new Set(
    target.mode === "edit" && target.destination.config.channel === "webhook"
      ? target.destination.config.headerNames
      : [],
  );
  return z
    .object({
      channel: z.enum(["discord", "slack", "webhook"]).nullable(),
      name: z.string().trim().min(1, "Enter a name.").max(128, "Use 128 characters or fewer."),
      url: z.string(),
      headers: z.array(z.object({ name: z.string(), value: z.string() })),
      enabled: z.boolean(),
    })
    .superRefine((values, ctx) => {
      if (!values.channel) {
        ctx.addIssue({ code: "custom", path: ["channel"], message: "Choose a channel." });
        return;
      }
      const keepsUrl = target.mode === "edit" && values.url.trim() === "";
      const problem = keepsUrl ? null : destinationUrlProblem(values.channel, values.url);
      if (problem) {
        ctx.addIssue({ code: "custom", path: ["url"], message: problem });
      }
      const seen = new Set<string>();
      values.headers.forEach((header, index) => {
        const nameProblem = headerNameProblem(header.name);
        if (nameProblem) {
          ctx.addIssue({ code: "custom", path: ["headers", index, "name"], message: nameProblem });
        }
        const key = header.name.trim().toLowerCase();
        if (seen.has(key)) {
          ctx.addIssue({
            code: "custom",
            path: ["headers", index, "name"],
            message: "Duplicate header.",
          });
        }
        seen.add(key);
        if (header.value === "" && !existingHeaders.has(header.name.trim())) {
          ctx.addIssue({
            code: "custom",
            path: ["headers", index, "value"],
            message: "Enter a value.",
          });
        }
      });
    });
}

type DestinationFormValues = z.infer<ReturnType<typeof formSchema>>;

function defaults(target: DestinationEditorTarget): DestinationFormValues {
  if (target.mode === "create") {
    return { channel: null, name: "", url: "", headers: [], enabled: true };
  }
  const { destination } = target;
  return {
    channel: destination.config.channel,
    name: destination.name,
    url: "",
    headers:
      destination.config.channel === "webhook"
        ? destination.config.headerNames.map((name) => ({ name, value: "" }))
        : [],
    enabled: destination.enabled,
  };
}

function nextConfig(
  channel: DestinationChannel,
  values: DestinationFormValues,
  previous: DestinationConfigSummary | null,
  signingSecret: string | null,
): DestinationConfigSummary {
  const hint = values.url.trim() === "" && previous ? previous.urlHint : urlHint(values.url);
  if (channel !== "webhook") {
    return { channel, urlHint: hint };
  }
  const previousSecretHint = previous?.channel === "webhook" ? previous.signingSecretHint : "";
  return {
    channel,
    urlHint: hint,
    headerNames: values.headers.map((header) => header.name.trim()),
    signingSecretHint: signingSecret ? secretHint(signingSecret) : previousSecretHint,
  };
}

export function DestinationPanel({
  target,
  isOpen,
  onClose,
  rules,
}: {
  target: DestinationEditorTarget;
  isOpen: boolean;
  onClose: () => void;
  rules: ReadonlyArray<Rule>;
}) {
  const [confirmDelete, setConfirmDelete] = useState(false);
  const { actions } = useNotificationPreview();
  const [revealedSecret, setRevealedSecret] = useState<string | null>(null);
  const [pendingSecret, setPendingSecret] = useState<string | null>(null);
  const form = useForm<DestinationFormValues>({
    resolver: standardSchemaResolver(formSchema(target)),
    defaultValues: defaults(target),
    mode: "onChange",
  });
  const { register, formState, setValue, control } = form;
  const headers = useFieldArray({ control, name: "headers" });
  const channel = form.watch("channel");
  const enabled = form.watch("enabled");
  const usedBy = target.mode === "edit" ? rulesUsingDestination(rules, target.destination.id) : [];
  const previous = target.mode === "edit" ? target.destination.config : null;

  const submit = form.handleSubmit((values) => {
    if (!values.channel) {
      return;
    }
    const id = target.mode === "edit" ? target.destination.id : newPreviewId("dst");
    const createdSecret =
      values.channel === "webhook" && target.mode === "create" ? generateSigningSecret() : null;
    const signingSecret = createdSecret ?? pendingSecret;
    actions.saveDestination({
      id,
      name: values.name.trim(),
      enabled: values.enabled,
      config: nextConfig(values.channel, values, previous, signingSecret),
    });
    toast.success(
      target.mode === "edit" ? "Destination updated in preview" : "Destination added to preview",
    );
    if (createdSecret) {
      setRevealedSecret(createdSecret);
      return;
    }
    onClose();
  });

  return (
    <SlidePanel isOpen={isOpen} onClose={onClose} widthClassName="w-160">
      <SlidePanelHeader className="items-center">
        <SlidePanelTitle>
          {target.mode === "edit" ? target.destination.name : "New destination"}
        </SlidePanelTitle>
        <SlidePanelCloseButton />
      </SlidePanelHeader>

      <SlidePanelContent>
        <form onSubmit={submit} className="flex h-full flex-col">
          <div className="flex flex-1 flex-col gap-6 overflow-y-auto px-6 pb-4 pt-1">
            {target.mode === "create" ? (
              <fieldset className="flex flex-col gap-2">
                <legend className="mb-2 text-sm text-gray-11">Channel</legend>
                <div className="grid gap-2">
                  {DESTINATION_CHANNELS.map((option) => (
                    <Item
                      key={option.channel}
                      variant="outline"
                      className={cn(
                        channel === option.channel && "border-gray-12 ring-2 ring-gray-5",
                      )}
                      render={
                        <button
                          type="button"
                          aria-pressed={channel === option.channel}
                          onClick={() =>
                            setValue("channel", option.channel, {
                              shouldDirty: true,
                              shouldValidate: true,
                            })
                          }
                        >
                          <ChannelMedia channel={option.channel} />
                          <ItemContent>
                            <ItemTitle>{option.title}</ItemTitle>
                            <ItemDescription>{option.description}</ItemDescription>
                          </ItemContent>
                        </button>
                      }
                    />
                  ))}
                </div>
                {formState.errors.channel?.message ? (
                  <p role="alert" className="text-xs text-error-11">
                    {formState.errors.channel.message}
                  </p>
                ) : null}
              </fieldset>
            ) : (
              <div className="flex items-center gap-3 rounded-lg border bg-raised px-4 py-3">
                <ChannelMedia channel={target.destination.config.channel} />
                <div className="flex min-w-0 flex-col">
                  <span className="text-sm text-gray-12">
                    {channelTitle(target.destination.config.channel)}
                  </span>
                  <span className="truncate font-mono text-xs text-gray-9">
                    {target.destination.config.urlHint}
                  </span>
                </div>
              </div>
            )}

            {channel ? (
              <>
                <FormInput
                  requirement="required"
                  label="Name"
                  description="Shown when you choose recipients for a rule."
                  placeholder={channel === "webhook" ? "On-call bridge" : "#incidents"}
                  error={formState.errors.name?.message}
                  {...register("name")}
                />
                <FormInput
                  requirement={target.mode === "create" ? "required" : undefined}
                  label={channel === "webhook" ? "Endpoint URL" : "Webhook URL"}
                  description={
                    target.mode === "edit"
                      ? "Leave empty to keep the current URL. Unkey encrypts it and only shows a hint."
                      : "Unkey encrypts this URL and only shows a hint after you save."
                  }
                  placeholder={URL_PLACEHOLDER[channel]}
                  error={formState.errors.url?.message}
                  autoComplete="off"
                  {...register("url")}
                />
                {channel === "webhook" ? (
                  <>
                    <fieldset className="flex flex-col gap-2">
                      <legend className="text-sm text-gray-11">Headers</legend>
                      <span className="text-xs text-gray-9">
                        Optional. Values are encrypted. Leave a saved header's value empty to keep
                        it.
                      </span>
                      {headers.fields.map((field, index) => (
                        <div
                          key={field.id}
                          className="grid grid-cols-[1fr_1fr_auto] items-start gap-2"
                        >
                          <FormInput
                            aria-label="Header name"
                            placeholder="X-Routing-Key"
                            error={formState.errors.headers?.[index]?.name?.message}
                            {...register(`headers.${index}.name`)}
                          />
                          <FormInput
                            aria-label="Header value"
                            type="password"
                            autoComplete="off"
                            placeholder="Value"
                            error={formState.errors.headers?.[index]?.value?.message}
                            {...register(`headers.${index}.value`)}
                          />
                          <Button
                            type="button"
                            variant="ghost"
                            className="mt-1 size-8 p-0"
                            aria-label="Remove header"
                            onClick={() => headers.remove(index)}
                          >
                            <IconTrashOutline12 />
                          </Button>
                        </div>
                      ))}
                      <Button
                        type="button"
                        variant="outline"
                        size="sm"
                        className="w-fit"
                        onClick={() => headers.append({ name: "", value: "" })}
                      >
                        <IconPlusOutline12 />
                        Add header
                      </Button>
                    </fieldset>
                    <div className="flex items-center justify-between gap-3 rounded-lg border bg-raised px-4 py-3">
                      <div className="flex flex-col">
                        <span className="text-sm text-gray-12">Signing secret</span>
                        <span className="text-xs text-gray-9">
                          {target.mode === "create"
                            ? "Unkey generates a secret when you save and shows it once."
                            : pendingSecret
                              ? `New secret ${secretHint(pendingSecret)} replaces the old one when you save.`
                              : `Requests are signed with ${previous?.channel === "webhook" ? previous.signingSecretHint : "a secret"}.`}
                        </span>
                      </div>
                      {target.mode === "edit" ? (
                        <Button
                          type="button"
                          variant="outline"
                          size="sm"
                          onClick={() => {
                            const secret = generateSigningSecret();
                            setPendingSecret(secret);
                            setRevealedSecret(secret);
                          }}
                        >
                          Rotate
                        </Button>
                      ) : null}
                    </div>
                  </>
                ) : null}

                <div className="flex items-center justify-between gap-3 rounded-lg border bg-raised px-4 py-3">
                  <div className="flex flex-col">
                    <span className="text-sm text-gray-12">Deliver to this destination</span>
                    <span className="text-xs text-gray-9">
                      Paused destinations stay on rules but receive nothing.
                    </span>
                  </div>
                  <Switch
                    aria-label="Deliver to this destination"
                    checked={enabled}
                    onCheckedChange={(next) => setValue("enabled", next, { shouldDirty: true })}
                  />
                </div>

                {target.mode === "edit" ? (
                  <div className="flex flex-col gap-1 rounded-lg border bg-raised px-4 py-3">
                    <span className="text-sm text-gray-12">
                      Used by {usedBy.length} {usedBy.length === 1 ? "rule" : "rules"}
                    </span>
                    <span className="text-xs text-gray-9">
                      {usedBy.length > 0
                        ? usedBy.map((rule) => rule.name).join(", ")
                        : "Add it to a rule to start receiving events."}
                    </span>
                  </div>
                ) : null}

                <div className="flex items-center justify-between gap-3 rounded-lg border border-dashed px-4 py-3">
                  <span className="text-xs text-gray-9">
                    Test messages need the delivery worker, which this preview does not include.
                  </span>
                  <Button type="button" variant="outline" size="sm" disabled>
                    Send test
                  </Button>
                </div>
              </>
            ) : null}
          </div>

          <div className="flex items-center justify-between gap-2 border-t px-6 py-4">
            <div>
              {target.mode === "edit" ? (
                <Button
                  type="button"
                  variant="outline"
                  color="danger"
                  onClick={() => setConfirmDelete(true)}
                >
                  Delete
                </Button>
              ) : null}
            </div>
            <div className="flex items-center gap-2">
              <Button type="button" variant="outline" onClick={onClose}>
                Cancel
              </Button>
              <Button type="submit" variant="primary" disabled={!channel}>
                {target.mode === "edit" ? "Save destination" : "Add destination"}
              </Button>
            </div>
          </div>
        </form>
      </SlidePanelContent>

      {target.mode === "edit" ? (
        <AlertDialog open={confirmDelete} onOpenChange={setConfirmDelete}>
          <AlertDialogContent>
            <AlertDialogHeader>
              <AlertDialogTitle>Delete {target.destination.name}?</AlertDialogTitle>
              <AlertDialogDescription>
                {usedBy.length > 0
                  ? `It is removed from ${usedBy.length} ${usedBy.length === 1 ? "rule" : "rules"}: ${usedBy.map((rule) => rule.name).join(", ")}. Other recipients on those rules are kept.`
                  : "No rules use this destination."}
              </AlertDialogDescription>
            </AlertDialogHeader>
            <AlertDialogFooter>
              <AlertDialogCancel>Keep destination</AlertDialogCancel>
              <AlertDialogAction
                color="danger"
                onClick={() => {
                  actions.deleteDestination(target.destination.id);
                  toast.success("Destination deleted from preview");
                  onClose();
                }}
              >
                Delete destination
              </AlertDialogAction>
            </AlertDialogFooter>
          </AlertDialogContent>
        </AlertDialog>
      ) : null}

      <AlertDialog
        open={revealedSecret !== null}
        onOpenChange={(open) => {
          if (!open) {
            const wasCreate = target.mode === "create";
            setRevealedSecret(null);
            if (wasCreate) {
              onClose();
            }
          }
        }}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Copy the signing secret</AlertDialogTitle>
            <AlertDialogDescription>
              Use it to verify that requests come from Unkey. You will not see it again. This
              preview secret is not stored anywhere.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <div className="flex min-w-0 items-center gap-2 rounded-lg border bg-grayA-2 px-3 py-2">
            <code className="min-w-0 flex-1 break-all font-mono text-xs text-gray-12">
              {revealedSecret}
            </code>
            <CopyButton value={revealedSecret ?? ""} variant="ghost" />
          </div>
          <AlertDialogFooter>
            <AlertDialogAction>Done</AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </SlidePanel>
  );
}
