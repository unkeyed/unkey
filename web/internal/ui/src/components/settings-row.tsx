"use client";

import * as React from "react";
import { cn } from "../lib/utils";
import { Button } from "./buttons/button";
import { InfoTooltip } from "./info-tooltip";
import { type SaveState, resolveGroupSave } from "./settings-save";
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

function SettingsGroupContent({
  className,
  children,
  pendingNote,
  ...props
}: React.ComponentProps<"div"> & {
  /** Shown beside Save only while the group has unsaved changes. */
  pendingNote?: React.ReactNode;
}) {
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
  useReportUnsavedChanges(values.some((member) => member.dirty));
  const save = resolveGroupSave(values);
  const blockedReasons = save.status === "blocked" ? save.reasons : [];
  const note =
    save.status === "ready" && save.submit.length < save.dirty
      ? `Saves ${save.submit.length} of ${save.dirty} changes`
      : save.status === "ready" || save.status === "blocked"
        ? pendingNote
        : null;

  const saveReady = () => {
    if (save.status !== "ready") {
      return;
    }
    for (const member of save.submit) {
      member.submit();
    }
  };

  return (
    <SettingsGroupContext.Provider value={context}>
      <div
        data-slot="settings-group-content"
        className={cn("@container border rounded-lg overflow-hidden bg-raised", className)}
        {...props}
      >
        <div className="divide-y divide-grayA-4">{children}</div>
        {values.length > 0 ? (
          <div className="border-t border-grayA-4 bg-grayA-2 px-5 py-3 flex items-center justify-end gap-3">
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
                  disabled={save.status !== "ready"}
                  loading={save.status === "saving"}
                  onClick={saveReady}
                >
                  Save changes
                </Button>
              </span>
            </InfoTooltip>
          </div>
        ) : null}
      </div>
    </SettingsGroupContext.Provider>
  );
}

SettingsGroupContent.displayName = "SettingsGroupContent";

function SettingsForm({
  dirty,
  saveState,
  onSubmit,
  autoSave = false,
  className,
  children,
}: {
  dirty: boolean;
  saveState: SaveState;
  onSubmit: React.FormEventHandler<HTMLFormElement>;
  /** Submits when focus leaves the form, and keeps the form out of the group's Save button. */
  autoSave?: boolean;
  className?: string;
  children: React.ReactNode;
}) {
  const formRef = React.useRef<HTMLFormElement>(null);

  useSettingsGroupMember({
    dirty: !autoSave && dirty,
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
      onBlur={(e) => {
        if (!autoSave || saveState.status !== "ready") {
          return;
        }
        const relatedTarget = e.relatedTarget instanceof Node ? e.relatedTarget : null;
        if (!e.currentTarget.contains(relatedTarget)) {
          e.currentTarget.requestSubmit();
        }
      }}
    >
      {children}
    </form>
  );
}

SettingsForm.displayName = "SettingsForm";

function SettingsRow({ className, ...props }: React.ComponentProps<"div">) {
  return (
    <div
      data-slot="settings-row"
      className={cn("flex flex-col gap-4 px-5 py-5 @2xl:flex-row", className)}
      {...props}
    />
  );
}

SettingsRow.displayName = "SettingsRow";

function SettingsRowHeader({ className, ...props }: React.ComponentProps<"div">) {
  return (
    <div
      data-slot="settings-row-header"
      className={cn("flex shrink-0 flex-col gap-1 @2xl:w-2/5", className)}
      {...props}
    />
  );
}

SettingsRowHeader.displayName = "SettingsRowHeader";

function SettingsRowTitle({ className, ...props }: React.ComponentProps<"div">) {
  return (
    <div
      data-slot="settings-row-title"
      className={cn("text-sm font-medium text-gray-12", className)}
      {...props}
    />
  );
}

SettingsRowTitle.displayName = "SettingsRowTitle";

function SettingsRowDescription({ className, ...props }: React.ComponentProps<"div">) {
  return (
    <div
      data-slot="settings-row-description"
      className={cn("text-xs leading-5 text-gray-11", className)}
      {...props}
    />
  );
}

SettingsRowDescription.displayName = "SettingsRowDescription";

function SettingsRowContent({ className, ...props }: React.ComponentProps<"div">) {
  return (
    <div
      data-slot="settings-row-content"
      className={cn("min-w-0 flex-1 [--setting-w:30rem]", className)}
      {...props}
    />
  );
}

SettingsRowContent.displayName = "SettingsRowContent";

function SettingsGroups({ className, ...props }: React.ComponentProps<"div">) {
  return (
    <div
      className={cn("mx-auto flex w-full max-w-[920px] flex-col gap-8 px-6 pt-6 pb-20", className)}
      {...props}
    />
  );
}

SettingsGroups.displayName = "SettingsGroups";

export { formSaveState, firstMatchingSaveState, type SaveState } from "./settings-save";
export {
  SettingsForm,
  SettingsGroup,
  SettingsGroupContent,
  SettingsGroupTitle,
  SettingsGroups,
  SettingsRow,
  SettingsRowContent,
  SettingsRowDescription,
  SettingsRowHeader,
  SettingsRowTitle,
};
