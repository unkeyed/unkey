import { type SaveState, SettingsRow, useSettingsGroupMember } from "@unkey/ui";
import type React from "react";
import { useRef } from "react";
import { useReportUnsavedChanges } from "../../prevent-leave-context";

type FormSettingCardProps = {
  title: string;
  description: React.ReactNode;
  requirement?: "required" | "optional";

  onSubmit: React.FormEventHandler<HTMLFormElement>;
  children: React.ReactNode;
  stickyHeader?: React.ReactNode;

  saveState: SaveState;

  ref?: React.Ref<HTMLFormElement>;
  contentRef?: React.Ref<HTMLDivElement>;
  autoSave?: boolean;
};

export const FormSettingCard = ({
  title,
  description,
  requirement,
  onSubmit,
  children,
  stickyHeader,
  saveState,
  ref,
  contentRef,
  autoSave,
}: FormSettingCardProps) => {
  const formRef = useRef<HTMLFormElement>(null);

  const dirty = !autoSave && saveState.status === "ready";
  useReportUnsavedChanges(dirty);
  useSettingsGroupMember({
    dirty,
    saving: saveState.status === "saving",
    submit: () => formRef.current?.requestSubmit(),
  });

  return (
    <form
      ref={(node) => {
        formRef.current = node;
        if (typeof ref === "function") {
          ref(node);
        } else if (ref) {
          ref.current = node;
        }
      }}
      onSubmit={onSubmit}
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
      <SettingsRow
        title={
          requirement ? (
            <span className="inline-flex items-center gap-2">
              {title}
              <span className="rounded-sm border bg-grayA-3 px-1 py-0.5 font-normal text-grayA-11 text-xs capitalize">
                {requirement}
              </span>
            </span>
          ) : (
            title
          )
        }
        description={description}
      >
        {stickyHeader}
        <div ref={contentRef} className="flex flex-col gap-2">
          {children}
        </div>
      </SettingsRow>
    </form>
  );
};

export function resolveSaveState(checks: ReadonlyArray<[boolean, SaveState]>): SaveState {
  for (const [condition, state] of checks) {
    if (condition) {
      return state;
    }
  }
  return { status: "ready" };
}
