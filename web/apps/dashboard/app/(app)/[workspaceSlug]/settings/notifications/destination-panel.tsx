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
  SettingsForm,
  SettingsPanel,
  SettingsRow,
  SettingsRowContent,
  SettingsRowDescription,
  SettingsRowHeader,
  SettingsRowTitle,
  cn,
  formSaveState,
  toast,
} from "@unkey/ui";
import { type FormEvent, useRef, useState } from "react";
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
  const savedRef = useRef(false);
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
  const dirty = formState.isDirty || pendingSecret !== null;
  const saveState = formSaveState({
    isSubmitting: formState.isSubmitting,
    isValid: formState.isValid,
    isDirty: dirty,
  });

  const submit = async (event: FormEvent<HTMLFormElement>) => {
    let saved = false;
    await form.handleSubmit((values) => {
      if (!values.channel || savedRef.current) {
        return;
      }
      const id = target.mode === "edit" ? target.destination.id : newPreviewId("dst");
      const createdSecret =
        values.channel === "webhook" && target.mode === "create" ? generateSigningSecret() : null;
      actions.saveDestination({
        id,
        name: values.name.trim(),
        enabled: values.enabled,
        config: nextConfig(values.channel, values, previous, createdSecret ?? pendingSecret),
      });
      savedRef.current = true;
      saved = true;
      form.reset(values);
      setPendingSecret(null);
      setRevealedSecret(createdSecret);
      toast.success(
        target.mode === "edit" ? "Destination updated in preview" : "Destination added to preview",
      );
      onClose();
    })(event);
    return saved;
  };

  return (
    <>
      <SettingsPanel
        isOpen={isOpen}
        onClose={onClose}
        onSaved={onClose}
        title={target.mode === "edit" ? target.destination.name : "New destination"}
        description="Preview only. Changes stay in this browser tab and do not send notifications."
      >
        <SettingsForm dirty={dirty} saveState={saveState} onSubmit={submit}>
          <SettingsRow>
            <SettingsRowHeader>
              <SettingsRowTitle>Channel</SettingsRowTitle>
            </SettingsRowHeader>
            <SettingsRowContent>
              {target.mode === "create" ? (
                <fieldset className="grid gap-2" aria-label="Channel">
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
                  {formState.errors.channel?.message ? (
                    <p role="alert" className="text-xs text-error-11">
                      {formState.errors.channel.message}
                    </p>
                  ) : null}
                </fieldset>
              ) : (
                <div className="flex items-center gap-3 text-sm text-gray-12">
                  <ChannelMedia channel={target.destination.config.channel} />
                  {channelTitle(target.destination.config.channel)}
                </div>
              )}
            </SettingsRowContent>
          </SettingsRow>

          {channel ? (
            <>
              <SettingsRow>
                <SettingsRowHeader>
                  <SettingsRowTitle>Name</SettingsRowTitle>
                  <SettingsRowDescription>
                    Shown when you choose recipients for a rule.
                  </SettingsRowDescription>
                </SettingsRowHeader>
                <SettingsRowContent>
                  <FormInput
                    requirement="required"
                    aria-label="Name"
                    placeholder={channel === "webhook" ? "On-call bridge" : "#incidents"}
                    error={formState.errors.name?.message}
                    {...register("name")}
                  />
                </SettingsRowContent>
              </SettingsRow>
              <SettingsRow>
                <SettingsRowHeader>
                  <SettingsRowTitle>
                    {channel === "webhook" ? "Endpoint URL" : "Webhook URL"}
                  </SettingsRowTitle>
                  <SettingsRowDescription>
                    {target.mode === "edit"
                      ? "Leave empty to keep the saved URL hint. This preview does not store the full URL."
                      : "Only a URL hint is kept in this preview."}
                    {previous ? (
                      <span className="mt-1 block break-all font-mono">{previous.urlHint}</span>
                    ) : null}
                  </SettingsRowDescription>
                </SettingsRowHeader>
                <SettingsRowContent>
                  <FormInput
                    requirement={target.mode === "create" ? "required" : undefined}
                    aria-label={channel === "webhook" ? "Endpoint URL" : "Webhook URL"}
                    placeholder={URL_PLACEHOLDER[channel]}
                    error={formState.errors.url?.message}
                    autoComplete="off"
                    {...register("url")}
                  />
                </SettingsRowContent>
              </SettingsRow>
              {channel === "webhook" ? (
                <>
                  <SettingsRow>
                    <SettingsRowHeader>
                      <SettingsRowTitle>Headers</SettingsRowTitle>
                      <SettingsRowDescription>
                        Optional. Leave a saved header's value empty to keep it. Only header names
                        are kept in this preview.
                      </SettingsRowDescription>
                    </SettingsRowHeader>
                    <SettingsRowContent className="flex flex-col gap-2">
                      {headers.fields.map((field, index) => (
                        <div
                          key={field.id}
                          className="grid grid-cols-[1fr_1fr_auto] items-start gap-2"
                        >
                          <FormInput
                            aria-label={`Header ${index + 1} name`}
                            placeholder="X-Routing-Key"
                            error={formState.errors.headers?.[index]?.name?.message}
                            {...register(`headers.${index}.name`)}
                          />
                          <FormInput
                            aria-label={`Header ${index + 1} value`}
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
                            aria-label={`Remove header ${index + 1}`}
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
                    </SettingsRowContent>
                  </SettingsRow>
                  <SettingsRow>
                    <SettingsRowHeader>
                      <SettingsRowTitle>Signing secret</SettingsRowTitle>
                      <SettingsRowDescription>
                        {target.mode === "create"
                          ? "Saving generates a preview secret and shows it once. Only its hint is kept."
                          : pendingSecret
                            ? `New secret ${secretHint(pendingSecret)} replaces the saved hint when you save.`
                            : `Saved preview hint: ${previous?.channel === "webhook" ? previous.signingSecretHint : "none"}.`}
                      </SettingsRowDescription>
                    </SettingsRowHeader>
                    {target.mode === "edit" ? (
                      <SettingsRowContent>
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
                      </SettingsRowContent>
                    ) : null}
                  </SettingsRow>
                </>
              ) : null}

              <SettingsRow>
                <SettingsRowHeader>
                  <SettingsRowTitle>Deliver to this destination</SettingsRowTitle>
                  <SettingsRowDescription>
                    Paused destinations stay on rules but receive nothing.
                  </SettingsRowDescription>
                </SettingsRowHeader>
                <SettingsRowContent>
                  <Switch
                    aria-label="Deliver to this destination"
                    checked={enabled}
                    onCheckedChange={(next) =>
                      setValue("enabled", next, { shouldDirty: true, shouldValidate: true })
                    }
                  />
                </SettingsRowContent>
              </SettingsRow>

              {target.mode === "edit" ? (
                <SettingsRow>
                  <SettingsRowHeader>
                    <SettingsRowTitle>
                      Used by {usedBy.length} {usedBy.length === 1 ? "rule" : "rules"}
                    </SettingsRowTitle>
                    <SettingsRowDescription>
                      {usedBy.length > 0
                        ? usedBy.map((rule) => rule.name).join(", ")
                        : "No rules use this destination."}
                    </SettingsRowDescription>
                  </SettingsRowHeader>
                  <SettingsRowContent>
                    <Button
                      type="button"
                      variant="outline"
                      color="danger"
                      onClick={() => setConfirmDelete(true)}
                    >
                      Delete destination
                    </Button>
                  </SettingsRowContent>
                </SettingsRow>
              ) : null}

              <SettingsRow>
                <SettingsRowHeader>
                  <SettingsRowTitle>Test delivery</SettingsRowTitle>
                  <SettingsRowDescription>
                    Test messages need the delivery worker, which this preview does not include.
                  </SettingsRowDescription>
                </SettingsRowHeader>
                <SettingsRowContent>
                  <Button type="button" variant="outline" size="sm" disabled>
                    Send test
                  </Button>
                </SettingsRowContent>
              </SettingsRow>
            </>
          ) : null}
        </SettingsForm>
      </SettingsPanel>

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
              <AlertDialogCancel type="button">Keep destination</AlertDialogCancel>
              <AlertDialogAction
                type="button"
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
            setRevealedSecret(null);
          }
        }}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Copy the signing secret</AlertDialogTitle>
            <AlertDialogDescription>
              This preview secret is not used for deliveries. You will not see it again. Only its
              hint is kept when you save.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <div className="flex min-w-0 items-center gap-2 rounded-lg border bg-grayA-2 px-3 py-2">
            <code className="min-w-0 flex-1 break-all font-mono text-xs text-gray-12">
              {revealedSecret}
            </code>
            <CopyButton type="button" value={revealedSecret ?? ""} variant="ghost" />
          </div>
          <AlertDialogFooter>
            <AlertDialogAction type="button">Done</AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </>
  );
}
