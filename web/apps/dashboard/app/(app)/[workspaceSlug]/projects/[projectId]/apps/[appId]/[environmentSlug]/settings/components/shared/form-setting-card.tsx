import { type SaveState, SettingsRow, useSettingsGroupMember } from "@unkey/ui";
import type React from "react";
import { useRef } from "react";
import { useReportUnsavedChanges } from "../../prevent-leave-context";

type SettingsFormProps = {
  onSubmit: React.FormEventHandler<HTMLFormElement>;
  children: React.ReactNode;
  saveState: SaveState;
  className?: string;
  ref?: React.Ref<HTMLFormElement>;
  autoSave?: boolean;
};

export function SettingsForm({
  onSubmit,
  children,
  saveState,
  className,
  ref,
  autoSave,
}: SettingsFormProps) {
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
      className={className}
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
      {children}
    </form>
  );
}

type FormSettingCardProps = Omit<SettingsFormProps, "className"> & {
  title: string;
  description: React.ReactNode;
  stickyHeader?: React.ReactNode;
  contentRef?: React.Ref<HTMLDivElement>;
};

export const FormSettingCard = ({
  title,
  description,
  children,
  stickyHeader,
  contentRef,
  ...formProps
}: FormSettingCardProps) => (
  <SettingsForm {...formProps}>
    <SettingsRow title={title} description={description}>
      {stickyHeader}
      <div ref={contentRef} className="flex flex-col gap-2">
        {children}
      </div>
    </SettingsRow>
  </SettingsForm>
);

export function resolveSaveState(checks: ReadonlyArray<[boolean, SaveState]>): SaveState {
  for (const [condition, state] of checks) {
    if (condition) {
      return state;
    }
  }
  return { status: "ready" };
}
