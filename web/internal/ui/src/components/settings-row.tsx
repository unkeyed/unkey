"use client";

import * as React from "react";
import { cn } from "../lib/utils";
import { Button } from "./buttons/button";
import { type GroupSave, type SaveState, resolveGroupSave } from "./settings-save";

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
  const group = React.useContext(SettingsGroupContext);
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

function SettingsGroupContent({ className, children, ...props }: React.ComponentProps<"div">) {
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
  const save = resolveGroupSave(values);

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
          <div className="flex items-center gap-3 border-t border-grayA-4 px-5 py-3">
            <SaveStatus save={save} />
            <Button
              variant="primary"
              size="sm"
              className="ml-auto px-3 disabled:border-transparent disabled:bg-grayA-3 disabled:text-grayA-9"
              disabled={save.status !== "ready"}
              loading={save.status === "saving"}
              onClick={saveReady}
            >
              Save changes
            </Button>
          </div>
        ) : null}
      </div>
    </SettingsGroupContext.Provider>
  );
}

SettingsGroupContent.displayName = "SettingsGroupContent";

function SaveStatus({ save }: { save: GroupSave<Member> }) {
  switch (save.status) {
    case "clean":
    case "saving":
      return null;
    case "blocked":
      return (
        <span className="truncate text-xs text-gray-11">
          {save.reasons.length > 0 ? save.reasons.join(" ") : "Can't save these changes yet"}
        </span>
      );
    case "ready":
      return save.submit.length < save.dirty ? (
        <span className="text-xs text-gray-11">
          Saves {save.submit.length} of {save.dirty} changes
        </span>
      ) : null;
  }
}

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
