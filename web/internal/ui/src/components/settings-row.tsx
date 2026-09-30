"use client";

import * as React from "react";
import { cn } from "../lib/utils";
import { Button } from "./buttons/button";

export type SaveState =
  | { status: "ready" }
  | { status: "disabled"; reason?: string }
  | { status: "saving" };

type Member = {
  dirty: boolean;
  saving: boolean;
  submit: () => void;
};

type GroupContext = {
  sync: (id: string, member: Member | null) => void;
};

const SettingsGroupContext = React.createContext<GroupContext | null>(null);

/**
 * Hands a row's dirty state and its submit to the group's one Save, so each
 * setting keeps its own form and its own mutation.
 */
function useSettingsGroupMember({ dirty, saving, submit }: Member) {
  const group = React.useContext(SettingsGroupContext);
  const id = React.useId();
  const submitRef = React.useRef(submit);
  submitRef.current = submit;

  React.useEffect(() => {
    group?.sync(id, { dirty, saving, submit: () => submitRef.current() });
    return () => group?.sync(id, null);
  }, [group, id, dirty, saving]);
}

function SettingsGroup({
  title,
  pendingNote,
  children,
}: {
  /** Omit when the page title already names the group. */
  title?: React.ReactNode;
  /** Shown beside Save only while the group has unsaved changes. */
  pendingNote?: React.ReactNode;
  children: React.ReactNode;
}) {
  const [members, setMembers] = React.useState<Readonly<Record<string, Member>>>({});

  const sync = React.useCallback((id: string, member: Member | null) => {
    setMembers((prev) => {
      if (!member) {
        const { [id]: _removed, ...rest } = prev;
        return rest;
      }
      return { ...prev, [id]: member };
    });
  }, []);

  const context = React.useMemo(() => ({ sync }), [sync]);
  const values = Object.values(members);
  const dirty = values.some((member) => member.dirty);
  const saving = values.some((member) => member.saving);

  const saveDirty = () => {
    for (const member of values) {
      if (member.dirty) {
        member.submit();
      }
    }
  };

  return (
    <SettingsGroupContext.Provider value={context}>
      <section className="flex flex-col gap-3">
        {title ? <h2 className="text-base font-medium text-gray-12">{title}</h2> : null}
        <div className="border rounded-lg overflow-hidden bg-raised">
          <div className="divide-y divide-grayA-4">{children}</div>
          <div className="border-t border-grayA-4 bg-grayA-2 px-5 py-3 flex items-center justify-end gap-3">
            {dirty && pendingNote ? (
              <span className="text-xs text-gray-11">{pendingNote}</span>
            ) : null}
            <Button
              variant="primary"
              size="sm"
              className="px-3"
              disabled={!dirty}
              loading={saving}
              onClick={saveDirty}
            >
              Save changes
            </Button>
          </div>
        </div>
      </section>
    </SettingsGroupContext.Provider>
  );
}

SettingsGroup.displayName = "SettingsGroup";

function SettingsRow({
  title,
  description,
  children,
}: {
  title: React.ReactNode;
  description?: React.ReactNode;
  children: React.ReactNode;
}) {
  return (
    <div className="flex flex-col lg:flex-row gap-4 px-5 py-5">
      <div className="lg:w-2/5 shrink-0 flex flex-col gap-1">
        <div className="text-sm font-medium text-gray-12">{title}</div>
        {description && <p className="text-xs leading-5 text-gray-11">{description}</p>}
      </div>
      <div className="flex-1 min-w-0 [--setting-w:30rem]">{children}</div>
    </div>
  );
}

SettingsRow.displayName = "SettingsRow";

function SettingsGroups({ className, ...props }: React.ComponentProps<"div">) {
  return (
    <div
      className={cn("mx-auto flex w-full max-w-[920px] flex-col gap-8 px-6 pt-6 pb-20", className)}
      {...props}
    />
  );
}

SettingsGroups.displayName = "SettingsGroups";

export { SettingsGroup, SettingsGroups, SettingsRow, useSettingsGroupMember };
