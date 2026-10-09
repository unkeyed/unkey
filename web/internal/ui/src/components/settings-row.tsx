"use client";

import { Collapsible } from "@base-ui/react/collapsible";
import { IconChevronDownOutline12 } from "@unkey/icons";
import * as React from "react";
import { cn } from "../lib/utils";
import { Button } from "./buttons/button";
import { InfoTooltip } from "./info-tooltip";
import { type GroupSave, type SaveState, resolveGroupSave } from "./settings-save";
import { useReportUnsavedChanges } from "./unsaved-changes";

type Member = {
  dirty: boolean;
  saveState: SaveState;
  submit: () => void;
};

type GroupContext = {
  update: (id: string, member: Member) => void;
  remove: (id: string) => void;
};

const SettingsGroupContext = React.createContext<GroupContext | null>(null);

const PendingNoteContext = React.createContext<React.ReactNode>(null);

const SHOWS_PENDING_NOTE: Record<GroupSave<Member>["status"], boolean> = {
  clean: false,
  saving: false,
  blocked: true,
  ready: true,
};

function groupNote(save: GroupSave<Member>, pendingNote: React.ReactNode): React.ReactNode {
  if (save.status === "ready" && save.submit.length < save.dirty) {
    return `Saves ${save.submit.length} of ${save.dirty} changes`;
  }
  return SHOWS_PENDING_NOTE[save.status] ? pendingNote : null;
}

function useSettingsGroupMember({ dirty, saveState, submit }: Member) {
  const group = React.use(SettingsGroupContext);
  if (!group) {
    throw new Error("A settings form must be rendered inside a SettingsGroupContent.");
  }
  const id = React.useId();
  const submitRef = React.useRef(submit);
  submitRef.current = submit;

  const { status } = saveState;
  const reason = saveState.status === "disabled" ? saveState.reason : undefined;
  const state = React.useMemo<SaveState>(
    () => (status === "disabled" ? { status, reason } : { status }),
    [status, reason],
  );

  React.useLayoutEffect(() => {
    group.update(id, { dirty, saveState: state, submit: () => submitRef.current() });
  }, [group, id, dirty, state]);

  React.useLayoutEffect(() => () => group.remove(id), [group, id]);
}

function SettingsGroup({ className, ...props }: React.ComponentProps<"section">) {
  return (
    <section
      data-slot="settings-group"
      className={cn("flex flex-col gap-3", className)}
      {...props}
    />
  );
}

SettingsGroup.displayName = "SettingsGroup";

function SettingsGroupTitle({ className, ...props }: React.ComponentProps<"h2">) {
  return (
    <h2
      data-slot="settings-group-title"
      className={cn("text-base font-medium text-gray-12", className)}
      {...props}
    />
  );
}

SettingsGroupTitle.displayName = "SettingsGroupTitle";

function useSettingsGroupMembers() {
  const [members, setMembers] = React.useState<ReadonlyMap<string, Member>>(new Map());

  const update = React.useCallback((id: string, member: Member) => {
    setMembers((prev) => new Map(prev).set(id, member));
  }, []);

  const remove = React.useCallback((id: string) => {
    setMembers((prev) => {
      const next = new Map(prev);
      next.delete(id);
      return next;
    });
  }, []);

  const context = React.useMemo(() => ({ update, remove }), [update, remove]);
  const values = [...members.values()];
  const dirty = values.some((member) => member.dirty || member.saveState.status === "saving");
  useReportUnsavedChanges(dirty);
  return { context, values, dirty };
}

function SettingsGroupBody({ values, children }: { values: Member[]; children: React.ReactNode }) {
  const pendingNote = React.use(PendingNoteContext);
  const save = resolveGroupSave(values);
  const blockedReasons = save.status === "blocked" ? save.reasons : [];
  const note = groupNote(save, pendingNote);

  const saveReady = () => {
    if (save.status !== "ready") {
      return;
    }
    for (const member of save.submit) {
      member.submit();
    }
  };

  return (
    <>
      <div className="divide-y divide-grayA-4">{children}</div>
      {values.length > 0 ? (
        <div className="border-t border-grayA-4 px-5 py-3 flex items-center justify-end gap-3">
          {note ? <span className="text-xs text-gray-11">{note}</span> : null}
          <InfoTooltip
            content={blockedReasons.join(" ")}
            disabled={blockedReasons.length === 0}
            asChild
          >
            <span>
              <Button
                variant="primary"
                size="sm"
                className="px-3"
                disabled={save.status === "clean" || save.status === "blocked"}
                loading={save.status === "saving"}
                onClick={saveReady}
              >
                Save
              </Button>
            </span>
          </InfoTooltip>
        </div>
      ) : null}
    </>
  );
}

function SettingsGroupContent({ className, children, ...props }: React.ComponentProps<"div">) {
  const { context, values } = useSettingsGroupMembers();
  return (
    <SettingsGroupContext.Provider value={context}>
      <div
        data-slot="settings-group-content"
        className={cn("@container border rounded-lg overflow-hidden bg-raised", className)}
        {...props}
      >
        <SettingsGroupBody values={values}>{children}</SettingsGroupBody>
      </div>
    </SettingsGroupContext.Provider>
  );
}

SettingsGroupContent.displayName = "SettingsGroupContent";

function SettingsGroupCollapsible({
  title,
  description,
  summary,
  className,
  children,
  ...props
}: Omit<React.ComponentProps<"div">, "title"> & {
  title: React.ReactNode;
  description?: React.ReactNode;
  /** Shown in the closed header. Replaced by "Unsaved changes" while a form in the card is dirty. */
  summary?: React.ReactNode;
}) {
  const { context, values, dirty } = useSettingsGroupMembers();
  return (
    <SettingsGroupContext.Provider value={context}>
      <Collapsible.Root
        data-slot="settings-group-collapsible"
        className={cn("@container border rounded-lg overflow-hidden bg-raised", className)}
        {...props}
      >
        <Collapsible.Trigger className="group flex w-full items-center gap-4 px-5 py-4 text-left transition-colors hover:bg-grayA-2 focus-visible:outline-hidden focus-visible:ring-3 focus-visible:ring-inset focus-visible:ring-gray-5">
          <span className="flex min-w-0 flex-1 flex-col gap-1">
            <span className="text-sm font-medium text-gray-12">{title}</span>
            {description ? (
              <span className="text-xs leading-5 text-gray-11">{description}</span>
            ) : null}
          </span>
          {dirty || summary ? (
            <span
              data-dirty={dirty}
              title={!dirty && typeof summary === "string" ? summary : undefined}
              className="min-w-0 max-w-1/2 truncate text-xs text-gray-11 data-[dirty=true]:text-warning-11 group-data-[panel-open]:hidden"
            >
              {dirty ? "Unsaved changes" : summary}
            </span>
          ) : null}
          <IconChevronDownOutline12 className="size-3 shrink-0 text-gray-10 transition-transform duration-200 ease-out group-data-[panel-open]:rotate-180" />
        </Collapsible.Trigger>
        <Collapsible.Panel
          keepMounted
          className="h-(--collapsible-panel-height) overflow-hidden border-t border-grayA-4 transition-[height] duration-200 ease-out data-[ending-style]:h-0 data-[starting-style]:h-0 motion-reduce:transition-none"
        >
          <SettingsGroupBody values={values}>{children}</SettingsGroupBody>
        </Collapsible.Panel>
      </Collapsible.Root>
    </SettingsGroupContext.Provider>
  );
}

SettingsGroupCollapsible.displayName = "SettingsGroupCollapsible";

function SettingsForm({
  dirty,
  saveState,
  onSubmit,
  className,
  children,
}: {
  dirty: boolean;
  saveState: SaveState;
  onSubmit: React.FormEventHandler<HTMLFormElement>;
  className?: string;
  children: React.ReactNode;
}) {
  const formRef = React.useRef<HTMLFormElement>(null);

  useSettingsGroupMember({
    dirty,
    saveState,
    submit: () => formRef.current?.requestSubmit(),
  });

  return (
    <form
      ref={formRef}
      className={className}
      onSubmit={(e) => {
        if (saveState.status !== "ready") {
          e.preventDefault();
          return;
        }
        onSubmit(e);
      }}
    >
      {children}
    </form>
  );
}

SettingsForm.displayName = "SettingsForm";

function SettingsRow({
  title,
  description,
  className,
  children,
  ...props
}: Omit<React.ComponentProps<"div">, "title"> & {
  title: React.ReactNode;
  description?: React.ReactNode;
}) {
  return (
    <div
      data-slot="settings-row"
      className={cn("flex flex-col gap-4 px-5 py-5 @2xl:flex-row", className)}
      {...props}
    >
      <div data-slot="settings-row-header" className="flex shrink-0 flex-col gap-1 @2xl:w-2/5">
        <div data-slot="settings-row-title" className="text-sm font-medium text-gray-12">
          {title}
        </div>
        {description ? (
          <div data-slot="settings-row-description" className="text-xs leading-5 text-gray-11">
            {description}
          </div>
        ) : null}
      </div>
      <div
        data-slot="settings-row-content"
        className="flex min-w-0 max-w-[30rem] flex-1 flex-col gap-1.5"
      >
        {children}
      </div>
    </div>
  );
}

SettingsRow.displayName = "SettingsRow";

function SettingsGroups({
  pendingNote = null,
  className,
  ...props
}: React.ComponentProps<"div"> & {
  /** Shown beside Save in every group while it has unsaved changes. */
  pendingNote?: React.ReactNode;
}) {
  return (
    <PendingNoteContext.Provider value={pendingNote}>
      <div
        className={cn(
          "mx-auto flex w-full max-w-[920px] flex-col gap-8 px-6 pt-6 pb-20",
          className,
        )}
        {...props}
      />
    </PendingNoteContext.Provider>
  );
}

SettingsGroups.displayName = "SettingsGroups";

export { formSaveState, firstMatchingSaveState, type SaveState } from "./settings-save";
export {
  SettingsForm,
  SettingsGroup,
  SettingsGroupCollapsible,
  SettingsGroupContent,
  SettingsGroupTitle,
  SettingsGroups,
  SettingsRow,
};
