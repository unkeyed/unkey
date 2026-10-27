"use client";

import { Switch } from "@/components/ui/switch";
import { creationRejected } from "@/lib/notifications/client";
import { notificationError } from "@/lib/notifications/use-notifications";
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
  FormSelect,
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
import {
  type DestinationFormValues,
  URL_PLACEHOLDER,
  destinationCreateInput,
  destinationFormSchema,
  destinationUpdateInput,
  generateSigningSecret,
} from "./destination-config";
import { DestinationTest } from "./destination-test";
import { useNotifications } from "./notification-context";
import { type Destination, type Rule, rulesUsingDestination } from "./notification-model";
import { ChannelMedia, DESTINATION_CHANNELS, channelTitle } from "./notification-ui";

export type DestinationEditorTarget =
  | { mode: "create" }
  | { mode: "edit"; destination: Destination };

function defaults(target: DestinationEditorTarget): DestinationFormValues {
  if (target.mode === "create") {
    return { channel: null, name: "", url: "", headersMode: "replace", headers: [], enabled: true };
  }
  const { destination } = target;
  return {
    channel: destination.channel,
    name: destination.name,
    url: "",
    headersMode: "keep",
    headers: [],
    enabled: destination.enabled,
  };
}

export function DestinationPanel({
  target,
  isOpen,
  onClose,
  onSave,
  rules,
}: {
  target: DestinationEditorTarget;
  isOpen: boolean;
  onClose: () => void;
  onSave?: (destination: Destination) => void;
  rules: ReadonlyArray<Rule>;
}) {
  const [confirmDelete, setConfirmDelete] = useState(false);
  const [deleting, setDeleting] = useState(false);
  const { actions, channels } = useNotifications();
  const [revealedSecret, setRevealedSecret] = useState<string | null>(null);
  const [pendingSecret, setPendingSecret] = useState<string | null>(null);
  const [idempotencyKey] = useState(() => crypto.randomUUID());
  const createdSecret = useRef<string | null>(null);
  const createAttempt = useRef<ReturnType<typeof destinationCreateInput> | null>(null);
  const [creationUnconfirmed, setCreationUnconfirmed] = useState(false);
  const savedRef = useRef(false);
  const form = useForm<DestinationFormValues>({
    resolver: standardSchemaResolver(destinationFormSchema(target.mode === "edit")),
    defaultValues: defaults(target),
    mode: "onChange",
  });
  const { register, formState, setValue, control } = form;
  const headers = useFieldArray({ control, name: "headers" });
  const channel = form.watch("channel");
  const enabled = form.watch("enabled");
  const headersMode = form.watch("headersMode");
  const usedBy = target.mode === "edit" ? rulesUsingDestination(rules, target.destination.id) : [];
  const previous = target.mode === "edit" ? target.destination : null;
  const dirty = formState.isDirty || pendingSecret !== null;
  const saveState = formSaveState({
    isSubmitting: formState.isSubmitting || deleting,
    isValid: formState.isValid,
    isDirty: dirty,
  });

  const submit = async (event: FormEvent<HTMLFormElement>) => {
    let saved = false;
    await form.handleSubmit(async (values) => {
      if (!values.channel || savedRef.current) {
        return;
      }
      try {
        if (values.channel === "webhook" && target.mode === "create") {
          createdSecret.current ??= generateSigningSecret();
        }
        let destination: Destination;
        if (target.mode === "create") {
          createAttempt.current ??= destinationCreateInput(
            values,
            idempotencyKey,
            createdSecret.current,
          );
          destination = await actions.createDestination(createAttempt.current);
        } else {
          destination = await actions.updateDestination(
            destinationUpdateInput(target.destination.id, values, pendingSecret),
          );
        }
        onSave?.(destination);
        savedRef.current = true;
        saved = true;
        form.reset({ ...values, url: "", headers: [], headersMode: "keep" });
        setRevealedSecret(
          values.channel === "webhook" ? (createdSecret.current ?? pendingSecret) : null,
        );
        createdSecret.current = null;
        createAttempt.current = null;
        setCreationUnconfirmed(false);
        setPendingSecret(null);
        toast.success(target.mode === "edit" ? "Channel updated" : "Channel connected");
        onClose();
      } catch (error) {
        if (target.mode === "create") {
          if (creationRejected(error) && !creationUnconfirmed) {
            createAttempt.current = null;
          } else {
            setCreationUnconfirmed(true);
          }
        }
        notificationError(error, "Couldn't save destination");
      }
    })(event);
    return saved;
  };

  return (
    <>
      <SettingsPanel
        isOpen={isOpen}
        onClose={() => {
          if (formState.isSubmitting || deleting) {
            return;
          }
          createdSecret.current = null;
          createAttempt.current = null;
          setPendingSecret(null);
          form.reset(defaults(target));
          onClose();
        }}
        onSaved={onClose}
        title={target.mode === "edit" ? target.destination.name : "Connect channel"}
        description="Connect a shared channel for notification routes. Credentials cannot be read after saving."
      >
        <SettingsForm dirty={dirty} saveState={saveState} onSubmit={submit}>
          {creationUnconfirmed ? (
            <p role="alert" className="text-xs text-warning-11">
              Creation could not be confirmed. Save again to retry the same request without creating
              a duplicate. Check connected channels before starting over.
            </p>
          ) : null}
          <fieldset
            className="contents"
            disabled={formState.isSubmitting || deleting || creationUnconfirmed}
          >
            <SettingsRow>
              <SettingsRowHeader>
                <SettingsRowTitle>Channel</SettingsRowTitle>
              </SettingsRowHeader>
              <SettingsRowContent>
                {target.mode === "create" ? (
                  <fieldset className="grid gap-2" aria-label="Channel">
                    {DESTINATION_CHANNELS.filter((option) => channels.includes(option.channel)).map(
                      (option) => (
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
                      ),
                    )}
                    {formState.errors.channel?.message ? (
                      <p role="alert" className="text-xs text-error-11">
                        {formState.errors.channel.message}
                      </p>
                    ) : null}
                  </fieldset>
                ) : (
                  <div className="flex items-center gap-3 text-sm text-gray-12">
                    <ChannelMedia channel={target.destination.channel} />
                    {channelTitle(target.destination.channel)}
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
                        ? "Leave empty to keep the saved URL. Only its origin is shown."
                        : "The full URL is write-only. Only its origin is shown after saving."}
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
                          Keep saved headers or replace the entire set, including every value. An
                          empty replacement removes all headers.
                          {previous?.headerNames.length ? (
                            <span className="mt-1 block">
                              Saved: {previous.headerNames.join(", ")}
                            </span>
                          ) : null}
                        </SettingsRowDescription>
                      </SettingsRowHeader>
                      <SettingsRowContent className="flex flex-col gap-2">
                        {target.mode === "edit" ? (
                          <FormSelect
                            label="Header changes"
                            value={headersMode}
                            options={[
                              { value: "keep", label: "Keep saved headers" },
                              { value: "replace", label: "Replace all headers" },
                            ]}
                            onValueChange={(value) => {
                              if (value === "keep" || value === "replace") {
                                setValue("headersMode", value, {
                                  shouldDirty: true,
                                  shouldValidate: true,
                                });
                              }
                            }}
                          />
                        ) : null}
                        {headersMode === "replace" ? (
                          <>
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
                              disabled={headers.fields.length >= 16}
                              onClick={() => headers.append({ name: "", value: "" })}
                            >
                              <IconPlusOutline12 />
                              Add header
                            </Button>
                          </>
                        ) : null}
                        {formState.errors.headers?.root?.message ||
                        formState.errors.headers?.message ? (
                          <p role="alert" className="text-xs text-error-11">
                            {formState.errors.headers.root?.message ??
                              formState.errors.headers.message}
                          </p>
                        ) : null}
                      </SettingsRowContent>
                    </SettingsRow>
                    <SettingsRow>
                      <SettingsRowHeader>
                        <SettingsRowTitle>Signing secret</SettingsRowTitle>
                        <SettingsRowDescription>
                          {target.mode === "create"
                            ? "Saving generates a signing secret and shows it once. Copy it to verify webhook signatures."
                            : pendingSecret
                              ? "A new secret will replace the current secret when you save. It will be shown after saving."
                              : "The saved secret cannot be read. Rotate it to generate a replacement."}
                        </SettingsRowDescription>
                      </SettingsRowHeader>
                      {target.mode === "edit" ? (
                        <SettingsRowContent>
                          <Button
                            type="button"
                            variant="outline"
                            size="sm"
                            disabled={formState.isSubmitting}
                            onClick={() => {
                              const secret = generateSigningSecret();
                              setPendingSecret(secret);
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
                        disabled={formState.isSubmitting}
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
                      Save changes before sending a test to this destination. A test sends a real
                      message.
                    </SettingsRowDescription>
                  </SettingsRowHeader>
                  <SettingsRowContent>
                    {target.mode === "edit" ? (
                      <DestinationTest
                        destinationId={target.destination.id}
                        disabled={dirty || formState.isSubmitting || deleting}
                        isOpen={isOpen}
                      />
                    ) : (
                      <Button type="button" variant="outline" size="sm" disabled>
                        Save before testing
                      </Button>
                    )}
                  </SettingsRowContent>
                </SettingsRow>
              </>
            ) : null}
          </fieldset>
        </SettingsForm>
      </SettingsPanel>

      {target.mode === "edit" ? (
        <AlertDialog
          open={confirmDelete}
          onOpenChange={(open) => {
            if (!deleting) {
              setConfirmDelete(open);
            }
          }}
        >
          <AlertDialogContent>
            <AlertDialogHeader>
              <AlertDialogTitle>Delete {target.destination.name}?</AlertDialogTitle>
              <AlertDialogDescription>
                {usedBy.length > 0
                  ? `Remove this destination from ${usedBy.length === 1 ? "this rule" : "these rules"} before deleting it: ${usedBy.map((rule) => rule.name).join(", ")}.`
                  : "No rules use this destination."}
              </AlertDialogDescription>
            </AlertDialogHeader>
            <AlertDialogFooter>
              <AlertDialogCancel type="button" disabled={deleting}>
                Keep destination
              </AlertDialogCancel>
              <Button
                type="button"
                color="danger"
                loading={deleting}
                disabled={deleting || usedBy.length > 0}
                onClick={async () => {
                  setDeleting(true);
                  try {
                    await actions.deleteDestination(target.destination.id);
                    toast.success("Destination deleted");
                    setConfirmDelete(false);
                    onClose();
                  } catch (error) {
                    notificationError(error, "Couldn't delete destination");
                  } finally {
                    setDeleting(false);
                  }
                }}
              >
                Delete destination
              </Button>
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
              This secret is now active. Copy it to verify webhook signatures. You will not be able
              to read it again.
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
