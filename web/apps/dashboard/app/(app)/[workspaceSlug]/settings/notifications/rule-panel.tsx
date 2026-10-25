"use client";

import {
  Multibox,
  MultiboxChip,
  MultiboxChipRemove,
  MultiboxChips,
  MultiboxContent,
  MultiboxEmpty,
  MultiboxInput,
  MultiboxItem,
  MultiboxList,
  MultiboxTrigger,
  useMultiboxAnchor,
} from "@/components/ui/multibox";
import { Switch } from "@/components/ui/switch";
import { standardSchemaResolver } from "@hookform/resolvers/standard-schema";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  Badge,
  Button,
  Checkbox,
  FormInput,
  FormSelect,
  SettingsForm,
  SettingsPanel,
  SettingsRow,
  SettingsRowContent,
  SettingsRowDescription,
  SettingsRowHeader,
  SettingsRowTitle,
  formSaveState,
  toast,
} from "@unkey/ui";
import { type FormEvent, type ReactNode, useState } from "react";
import { Controller, useForm } from "react-hook-form";
import { z } from "zod";
import {
  type Destination,
  type Member,
  NAMED_AUDIENCES,
  type NamedAudience,
  type RecipientTarget,
  type ResourceScope,
  type Rule,
  WORKSPACE_WIDE,
  audienceMembers,
  ruleReach,
} from "./notification-model";
import {
  NOTIFICATION_CATEGORIES,
  NOTIFICATION_TYPES,
  findNotificationType,
} from "./notification-types";
import { ChannelIcon } from "./notification-ui";
import { useNotificationPreview } from "./preview-context";
import { isSampleMember, newPreviewId } from "./preview-store";

const ALL = "all";

const ruleFormSchema = z
  .object({
    name: z.string().trim().min(1, "Enter a name.").max(128, "Use 128 characters or fewer."),
    notificationType: z.string().min(1, "Choose an event."),
    projectId: z.string(),
    appId: z.string(),
    environmentId: z.string(),
    audiences: z.array(z.number()),
    memberIds: z.array(z.string()),
    destinationIds: z.array(z.string()),
    enabled: z.boolean(),
  })
  .refine(
    (values) =>
      values.audiences.length + values.memberIds.length + values.destinationIds.length > 0,
    { path: ["audiences"], message: "Choose at least one recipient." },
  );

type RuleFormValues = z.infer<typeof ruleFormSchema>;

export type RuleEditorTarget =
  | { mode: "edit"; rule: Rule }
  | { mode: "create"; notificationType?: string; scope?: ResourceScope };

function toFormValues(target: RuleEditorTarget): RuleFormValues {
  if (target.mode === "create") {
    const scope = target.scope ?? WORKSPACE_WIDE;
    return {
      name: "",
      notificationType: target.notificationType ?? "",
      projectId: scope.projectId ?? ALL,
      appId: scope.appId ?? ALL,
      environmentId: scope.environmentId ?? ALL,
      audiences: [],
      memberIds: [],
      destinationIds: [],
      enabled: true,
    };
  }
  const { rule } = target;
  return {
    name: rule.name,
    notificationType: rule.notificationType,
    projectId: rule.scope.projectId ?? ALL,
    appId: rule.scope.appId ?? ALL,
    environmentId: rule.scope.environmentId ?? ALL,
    audiences: rule.targets.flatMap((t) => (t.kind === "audience" ? [t.audience] : [])),
    memberIds: rule.targets.flatMap((t) => (t.kind === "member" ? [t.userId] : [])),
    destinationIds: rule.targets.flatMap((t) =>
      t.kind === "destination" ? [t.destinationId] : [],
    ),
    enabled: rule.enabled,
  };
}

function isNamedAudience(value: number): value is NamedAudience {
  return NAMED_AUDIENCES.some((option) => option.audience === value);
}

function toRule(id: string, values: RuleFormValues): Rule {
  const scoped = findNotificationType(values.notificationType)?.resourceScoped ?? false;
  const targets: RecipientTarget[] = [
    ...values.audiences
      .filter(isNamedAudience)
      .map((audience) => ({ kind: "audience" as const, audience })),
    ...values.memberIds.map((userId) => ({ kind: "member" as const, userId })),
    ...values.destinationIds.map((destinationId) => ({
      kind: "destination" as const,
      destinationId,
    })),
  ];
  const pick = (value: string) => (scoped && value !== ALL ? value : null);
  return {
    id,
    name: values.name.trim(),
    notificationType: values.notificationType,
    scope: {
      projectId: pick(values.projectId),
      appId: values.projectId === ALL ? null : pick(values.appId),
      environmentId: values.appId === ALL ? null : pick(values.environmentId),
    },
    enabled: values.enabled,
    targets,
  };
}

function Section({
  title,
  description,
  children,
}: {
  title: string;
  description?: ReactNode;
  children: ReactNode;
}) {
  return (
    <SettingsRow>
      <SettingsRowHeader>
        <SettingsRowTitle>{title}</SettingsRowTitle>
        {description ? <SettingsRowDescription>{description}</SettingsRowDescription> : null}
      </SettingsRowHeader>
      <SettingsRowContent>
        <div className="flex flex-col gap-3">{children}</div>
      </SettingsRowContent>
    </SettingsRow>
  );
}

type ResourceOptions = {
  projects: ReadonlyArray<{
    id: string;
    name: string;
    apps: ReadonlyArray<{ id: string; name: string }>;
  }>;
  environments: ReadonlyArray<{ id: string; appId: string; slug: string }>;
};

export function RulePanel({
  target,
  isOpen,
  onClose,
  members,
  destinations,
  resources,
  currentUserId,
}: {
  target: RuleEditorTarget;
  isOpen: boolean;
  onClose: () => void;
  members: ReadonlyArray<Member>;
  destinations: ReadonlyArray<Destination>;
  resources: ResourceOptions;
  currentUserId: string | null;
}) {
  const [confirmDelete, setConfirmDelete] = useState(false);
  const { actions, preferences } = useNotificationPreview();
  const form = useForm<RuleFormValues>({
    resolver: standardSchemaResolver(ruleFormSchema),
    defaultValues: toFormValues(target),
    mode: "onChange",
  });
  const { control, register, formState, setValue } = form;
  const values = form.watch();
  const typeDefinition = findNotificationType(values.notificationType);
  const project = resources.projects.find((candidate) => candidate.id === values.projectId);
  const appEnvironments = resources.environments.filter((env) => env.appId === values.appId);

  const ruleId = target.mode === "edit" ? target.rule.id : "draft";
  const reach = ruleReach(toRule(ruleId, values), { members, destinations, preferences });
  const emailCount = reach.email.filter((recipient) => recipient.status === "send").length;
  const destinationCount = reach.destinations.filter((d) => d.status === "send").length;

  const memberLabel = (userId: string) => {
    const member = members.find((candidate) => candidate.userId === userId);
    if (!member) {
      return "Former member";
    }
    return userId === currentUserId ? `${member.name} (you)` : member.name;
  };

  const submit = async (event: FormEvent<HTMLFormElement>) => {
    let saved = false;
    await form.handleSubmit((submitted) => {
      const id = target.mode === "edit" ? target.rule.id : newPreviewId("rule");
      actions.saveRule(toRule(id, submitted));
      form.reset(submitted);
      toast.success(target.mode === "edit" ? "Rule updated in preview" : "Rule added to preview");
      saved = true;
    })(event);
    return saved;
  };

  return (
    <SettingsPanel
      isOpen={isOpen}
      onClose={onClose}
      onSaved={onClose}
      title={target.mode === "edit" ? "Edit rule" : "New rule"}
      description="Other matching rules still apply. Each recipient gets one message."
    >
      <SettingsForm
        onSubmit={submit}
        dirty={formState.isDirty}
        saveState={formSaveState(formState)}
      >
        <Section title="Event">
          <FormInput
            requirement="required"
            label="Rule name"
            placeholder="Production deploy failures"
            error={formState.errors.name?.message}
            {...register("name")}
          />
          <Controller
            control={control}
            name="notificationType"
            render={({ field, fieldState }) => (
              <FormSelect
                requirement="required"
                label="Notify about"
                placeholder="Choose an event"
                value={field.value}
                onValueChange={field.onChange}
                error={fieldState.error?.message}
                description={typeDefinition?.description}
                options={NOTIFICATION_CATEGORIES.flatMap(({ category, label }) =>
                  NOTIFICATION_TYPES.filter((type) => type.category === category).map((type) => ({
                    value: type.type,
                    label: (
                      <span className="flex items-center gap-2">
                        <span className="text-gray-9">{label} /</span>
                        {type.label}
                      </span>
                    ),
                  })),
                )}
              />
            )}
          />
        </Section>

        {typeDefinition?.resourceScoped ? (
          <Section
            title="Where"
            description="All includes matching resources created later. This rule adds recipients; it does not override broader rules."
          >
            <div className="grid gap-3 sm:grid-cols-3">
              <FormSelect
                label="Project"
                value={values.projectId}
                onValueChange={(next) => {
                  setValue("projectId", next, { shouldDirty: true });
                  setValue("appId", ALL);
                  setValue("environmentId", ALL);
                }}
                options={[
                  { value: ALL, label: "All projects" },
                  ...resources.projects.map((p) => ({ value: p.id, label: p.name })),
                ]}
              />
              <FormSelect
                label="App"
                value={values.appId}
                disabled={!project}
                onValueChange={(next) => {
                  setValue("appId", next, { shouldDirty: true });
                  setValue("environmentId", ALL);
                }}
                options={[
                  { value: ALL, label: "All apps" },
                  ...(project?.apps ?? []).map((app) => ({ value: app.id, label: app.name })),
                ]}
              />
              <FormSelect
                label="Environment"
                value={values.environmentId}
                disabled={values.appId === ALL}
                onValueChange={(next) => setValue("environmentId", next, { shouldDirty: true })}
                options={[
                  { value: ALL, label: "All environments" },
                  ...appEnvironments.map((env) => ({ value: env.id, label: env.slug })),
                ]}
              />
            </div>
          </Section>
        ) : typeDefinition ? (
          <p className="px-1 text-xs text-gray-9">
            {typeDefinition.label} events apply to the whole workspace.
          </p>
        ) : null}

        <Section
          title="Email"
          description="Unkey sends email to each member's account address. Members can turn off email per event in their own preferences."
        >
          <Controller
            control={control}
            name="audiences"
            render={({ field }) => (
              <div className="flex flex-col gap-2">
                {NAMED_AUDIENCES.map((option) => {
                  const count = audienceMembers(members, option.audience).length;
                  const checked = field.value.includes(option.audience);
                  return (
                    <label
                      key={option.audience}
                      htmlFor={`rule-audience-${option.audience}`}
                      className="flex cursor-pointer items-start gap-3 rounded-lg border px-3 py-2.5 hover:bg-grayA-2"
                    >
                      <Checkbox
                        id={`rule-audience-${option.audience}`}
                        className="mt-0.5"
                        checked={checked}
                        onCheckedChange={(next) =>
                          field.onChange(
                            next
                              ? [...field.value, option.audience]
                              : field.value.filter((value) => value !== option.audience),
                          )
                        }
                      />
                      <span className="flex min-w-0 flex-1 flex-col">
                        <span className="text-sm text-gray-12">{option.label}</span>
                        <span className="text-xs text-gray-9">{option.description}</span>
                      </span>
                      <span className="shrink-0 text-xs text-gray-9 tabular-nums">
                        {count} {count === 1 ? "member" : "members"}
                      </span>
                    </label>
                  );
                })}
              </div>
            )}
          />
          <MemberPicker
            value={values.memberIds}
            onChange={(next) =>
              setValue("memberIds", next, { shouldDirty: true, shouldValidate: true })
            }
            members={members}
            getLabel={memberLabel}
          />
        </Section>

        <Section
          title="Destinations"
          description="Shared channels and endpoints. Manage them in the Destinations tab."
        >
          {destinations.length === 0 ? (
            <p className="text-xs text-gray-9">No destinations yet.</p>
          ) : (
            <Controller
              control={control}
              name="destinationIds"
              render={({ field }) => (
                <div className="flex flex-col gap-2">
                  {destinations.map((destination) => (
                    <label
                      key={destination.id}
                      htmlFor={`rule-destination-${destination.id}`}
                      className="flex cursor-pointer items-center gap-3 rounded-lg border px-3 py-2.5 hover:bg-grayA-2"
                    >
                      <Checkbox
                        id={`rule-destination-${destination.id}`}
                        checked={field.value.includes(destination.id)}
                        onCheckedChange={(next) =>
                          field.onChange(
                            next
                              ? [...field.value, destination.id]
                              : field.value.filter((value) => value !== destination.id),
                          )
                        }
                      />
                      <ChannelIcon channel={destination.config.channel} className="size-4" />
                      <span className="min-w-0 flex-1 truncate text-sm text-gray-12">
                        {destination.name}
                      </span>
                      {destination.enabled ? null : <Badge variant="secondary">Paused</Badge>}
                    </label>
                  ))}
                </div>
              )}
            />
          )}
        </Section>

        {formState.errors.audiences?.message ? (
          <p role="alert" className="px-1 text-xs text-error-11">
            {formState.errors.audiences.message}
          </p>
        ) : null}

        <div className="flex items-center justify-between gap-3 rounded-lg border bg-raised px-4 py-3">
          <div className="flex flex-col">
            <span className="text-sm text-gray-12">Rule is active</span>
            <span className="text-xs text-gray-9">Paused rules keep their recipients.</span>
          </div>
          <Controller
            control={control}
            name="enabled"
            render={({ field }) => (
              <Switch
                aria-label="Rule is active"
                checked={field.value}
                onCheckedChange={field.onChange}
              />
            )}
          />
        </div>
        <Section title="Recipients">
          <output className="text-xs text-gray-11" aria-live="polite">
            Emails {emailCount} {emailCount === 1 ? "person" : "people"}, posts to{" "}
            {destinationCount} {destinationCount === 1 ? "destination" : "destinations"}
          </output>
        </Section>
      </SettingsForm>
      {target.mode === "edit" ? (
        <div className="mt-6 border-t pt-4">
          <Button
            type="button"
            variant="outline"
            color="danger"
            onClick={() => setConfirmDelete(true)}
          >
            Delete rule
          </Button>
        </div>
      ) : null}

      {target.mode === "edit" ? (
        <AlertDialog open={confirmDelete} onOpenChange={setConfirmDelete}>
          <AlertDialogContent>
            <AlertDialogHeader>
              <AlertDialogTitle>Delete {target.rule.name}?</AlertDialogTitle>
              <AlertDialogDescription>
                Recipients on this rule stop getting these events unless another rule includes them.
              </AlertDialogDescription>
            </AlertDialogHeader>
            <AlertDialogFooter>
              <AlertDialogCancel>Keep rule</AlertDialogCancel>
              <AlertDialogAction
                color="danger"
                onClick={() => {
                  actions.deleteRule(target.rule.id);
                  toast.success("Rule deleted from preview");
                  onClose();
                }}
              >
                Delete rule
              </AlertDialogAction>
            </AlertDialogFooter>
          </AlertDialogContent>
        </AlertDialog>
      ) : null}
    </SettingsPanel>
  );
}

function MemberPicker({
  value,
  onChange,
  members,
  getLabel,
}: {
  value: string[];
  onChange: (next: string[]) => void;
  members: ReadonlyArray<Member>;
  getLabel: (userId: string) => string;
}) {
  const anchor = useMultiboxAnchor();
  return (
    <div className="flex flex-col gap-1.5">
      <span className="text-sm text-gray-11">Specific members</span>
      <Multibox
        items={members.map((member) => member.userId)}
        value={value}
        onValueChange={onChange}
        itemToStringLabel={getLabel}
      >
        <MultiboxChips ref={anchor}>
          {value.map((userId) => (
            <MultiboxChip key={userId}>
              {getLabel(userId)}
              <MultiboxChipRemove />
            </MultiboxChip>
          ))}
          <MultiboxInput
            aria-label="Search members"
            placeholder={value.length === 0 ? "Add members by name" : "Search"}
          />
          <MultiboxTrigger />
        </MultiboxChips>
        <MultiboxContent anchor={anchor}>
          <MultiboxEmpty>No members found.</MultiboxEmpty>
          <MultiboxList>
            {(userId: string) => {
              const member = members.find((candidate) => candidate.userId === userId);
              return (
                <MultiboxItem key={userId} value={userId}>
                  <span className="flex min-w-0 flex-1 items-center justify-between gap-2">
                    <span className="truncate">{getLabel(userId)}</span>
                    <span className="flex items-center gap-1.5 text-xs text-gray-9">
                      {member?.email}
                      {isSampleMember(userId) ? <Badge size="sm">Sample</Badge> : null}
                    </span>
                  </span>
                </MultiboxItem>
              );
            }}
          </MultiboxList>
        </MultiboxContent>
      </Multibox>
    </div>
  );
}
