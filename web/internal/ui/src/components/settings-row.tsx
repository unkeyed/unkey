"use client";

import { Collapsible } from "@base-ui/react/collapsible";
import { IconChevronDownOutline12 } from "@unkey/icons";
import * as React from "react";
import { cn } from "../lib/utils";
import { Button } from "./buttons/button";
import { InfoTooltip } from "./info-tooltip";
import { type GroupSave, type SaveState, resolveGroupSave } from "./settings-save";
import { Skeleton } from "./skeleton";
import {
  SlidePanel,
  SlidePanelClose,
  SlidePanelCloseButton,
  SlidePanelContent,
  SlidePanelDescription,
  SlidePanelFooter,
  SlidePanelHeader,
  SlidePanelTitle,
} from "./slide-panel";
import { useReportUnsavedChanges } from "./unsaved-changes";

type Member = {
  dirty: boolean;
  saveState: SaveState;
  submit: () => Promise<boolean>;
};

type GroupContext = {
  update: (id: string, member: Member) => void;
  remove: (id: string) => void;
};

const SettingsGroupContext = React.createContext<GroupContext | null>(null);

const PendingNoteContext = React.createContext<React.ReactNode>(null);

type SettingsLayout = "rows" | "stacked";

const SettingsLayoutContext = React.createContext<SettingsLayout>("rows");

const LAYOUT_CLASS: Record<
  SettingsLayout,
  Record<"shell" | "body" | "row" | "header" | "title" | "content" | "description", string>
> = {
  rows: {
    shell: "@container rounded-lg overflow-hidden bg-raised shadow-raised",
    body: "divide-y",
    row: "flex flex-col gap-4 px-5 py-5 @2xl:flex-row",
    header: "flex shrink-0 flex-col gap-1 @2xl:w-2/5",
    title: "text-sm font-medium text-gray-12",
    content: "min-w-0 flex-1 [--setting-w:30rem]",
    description: "text-xs leading-5 text-gray-11",
  },
  stacked: {
    shell: "@container",
    body: "flex flex-col gap-5",
    row: "flex flex-col gap-1.5",
    header: "contents",
    title: "order-1 text-sm font-medium text-gray-12",
    content: "order-2 min-w-0 [--setting-w:100%]",
    description: "order-3 text-xs leading-5 text-gray-11",
  },
};

function useLayoutClass(slot: keyof (typeof LAYOUT_CLASS)["rows"]): string {
  return LAYOUT_CLASS[React.use(SettingsLayoutContext)][slot];
}

const SHOWS_PENDING_NOTE: Record<GroupSave<Member>["status"], boolean> = {
  clean: false,
  saving: false,
  blocked: true,
  ready: true,
};

function groupNote(save: GroupSave<Member>, pendingNote: React.ReactNode): React.ReactNode {
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

function GroupSaveButton({
  save,
  size,
  onSaved,
}: {
  save: GroupSave<Member>;
  size: "sm" | "md";
  onSaved?: () => void;
}) {
  const blockedReasons = save.status === "blocked" ? save.reasons : [];
  const saveReady = async () => {
    if (save.status !== "ready") {
      return;
    }
    const results = await Promise.all(save.submit.map((member) => member.submit()));
    if (save.complete && results.every(Boolean)) {
      onSaved?.();
    }
  };

  return (
    <InfoTooltip content={blockedReasons.join(" ")} disabled={blockedReasons.length === 0} asChild>
      <span>
        <Button
          variant="primary"
          size={size}
          className="px-3"
          disabled={save.status === "clean" || save.status === "blocked"}
          loading={save.status === "saving"}
          onClick={saveReady}
        >
          Save
        </Button>
      </span>
    </InfoTooltip>
  );
}

function SettingsGroupBody({ values, children }: { values: Member[]; children: React.ReactNode }) {
  const pendingNote = React.use(PendingNoteContext);
  const save = resolveGroupSave(values);
  const note = groupNote(save, pendingNote);

  return (
    <>
      <div className={useLayoutClass("body")}>{children}</div>
      {values.length > 0 ? (
        <div className="border-t px-5 py-3 flex items-center justify-end gap-3">
          {note ? <span className="text-xs text-gray-11">{note}</span> : null}
          <GroupSaveButton save={save} size="sm" />
        </div>
      ) : null}
    </>
  );
}

function SettingsGroupShell({ className, ...props }: React.ComponentProps<"div">) {
  return <div className={cn(useLayoutClass("shell"), className)} {...props} />;
}

function SettingsGroupContent({ className, children, ...props }: React.ComponentProps<"div">) {
  const { context, values } = useSettingsGroupMembers();
  return (
    <SettingsGroupContext.Provider value={context}>
      <SettingsGroupShell data-slot="settings-group-content" className={className} {...props}>
        <SettingsGroupBody values={values}>{children}</SettingsGroupBody>
      </SettingsGroupShell>
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
        render={<SettingsGroupShell />}
        data-slot="settings-group-collapsible"
        className={className}
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
          className="h-(--collapsible-panel-height) overflow-hidden border-t transition-[height] duration-200 ease-out data-[ending-style]:h-0 data-[starting-style]:h-0 motion-reduce:transition-none"
        >
          <SettingsGroupBody values={values}>{children}</SettingsGroupBody>
        </Collapsible.Panel>
      </Collapsible.Root>
    </SettingsGroupContext.Provider>
  );
}

SettingsGroupCollapsible.displayName = "SettingsGroupCollapsible";

type SettingsPanelProps = {
  isOpen: boolean;
  onClose: () => void;
  /** Runs after Save when every changed row saved. */
  onSaved: () => void;
  title: React.ReactNode;
  description?: React.ReactNode;
  children: React.ReactNode;
};

function SettingsPanel({ isOpen, onClose, ...props }: SettingsPanelProps) {
  return (
    <SlidePanel isOpen={isOpen} onClose={onClose}>
      <SettingsPanelBody {...props} />
    </SlidePanel>
  );
}

function SettingsPanelBody({
  onSaved,
  title,
  description,
  children,
}: Omit<SettingsPanelProps, "isOpen" | "onClose">) {
  const { context, values } = useSettingsGroupMembers();
  const save = resolveGroupSave(values);
  return (
    <SettingsGroupContext.Provider value={context}>
      <SlidePanelHeader>
        <div className="flex flex-col gap-0.5">
          <SlidePanelTitle>{title}</SlidePanelTitle>
          {description ? <SlidePanelDescription>{description}</SlidePanelDescription> : null}
        </div>
        <SlidePanelCloseButton className="mt-0.5" />
      </SlidePanelHeader>
      <SlidePanelContent className="overflow-y-auto px-6 pt-4 pb-8">
        <SettingsLayoutContext.Provider value="stacked">
          <div data-slot="settings-panel-content" className={LAYOUT_CLASS.stacked.shell}>
            <div className={LAYOUT_CLASS.stacked.body}>{children}</div>
          </div>
        </SettingsLayoutContext.Provider>
      </SlidePanelContent>
      <SlidePanelFooter className="flex items-center justify-end gap-3">
        <SlidePanelClose
          disabled={save.status === "saving"}
          render={<Button type="button" variant="outline" size="md" />}
        >
          Cancel
        </SlidePanelClose>
        <GroupSaveButton save={save} size="md" onSaved={onSaved} />
      </SlidePanelFooter>
    </SettingsGroupContext.Provider>
  );
}

SettingsPanel.displayName = "SettingsPanel";

function SettingsForm({
  dirty,
  saveState,
  onSubmit,
  className,
  children,
}: {
  dirty: boolean;
  saveState: SaveState;
  onSubmit: (event: React.FormEvent<HTMLFormElement>) => void | Promise<void> | Promise<boolean>;
  className?: string;
  children: React.ReactNode;
}) {
  const formRef = React.useRef<HTMLFormElement>(null);
  const resultRef = React.useRef<Promise<boolean> | null>(null);

  useSettingsGroupMember({
    dirty,
    saveState,
    submit: () => {
      resultRef.current = null;
      formRef.current?.requestSubmit();
      return resultRef.current ?? Promise.resolve(false);
    },
  });

  return (
    <form
      ref={formRef}
      className={cn(useLayoutClass("body"), className)}
      onSubmit={(e) => {
        if (saveState.status !== "ready") {
          e.preventDefault();
          return;
        }
        resultRef.current = Promise.resolve(onSubmit(e)).then(
          (result) => result !== false,
          () => false,
        );
      }}
    >
      {children}
    </form>
  );
}

SettingsForm.displayName = "SettingsForm";

function SettingsRow({ className, ...props }: React.ComponentProps<"div">) {
  return (
    <div data-slot="settings-row" className={cn(useLayoutClass("row"), className)} {...props} />
  );
}

SettingsRow.displayName = "SettingsRow";

function SettingsRowHeader({ className, ...props }: React.ComponentProps<"div">) {
  return (
    <div
      data-slot="settings-row-header"
      className={cn(useLayoutClass("header"), className)}
      {...props}
    />
  );
}

SettingsRowHeader.displayName = "SettingsRowHeader";

function SettingsRowTitle({ className, ...props }: React.ComponentProps<"div">) {
  return (
    <div
      data-slot="settings-row-title"
      className={cn(useLayoutClass("title"), className)}
      {...props}
    />
  );
}

SettingsRowTitle.displayName = "SettingsRowTitle";

function SettingsRowDescription({ className, ...props }: React.ComponentProps<"div">) {
  return (
    <div
      data-slot="settings-row-description"
      className={cn(useLayoutClass("description"), className)}
      {...props}
    />
  );
}

SettingsRowDescription.displayName = "SettingsRowDescription";

function SettingsRowContent({ className, ...props }: React.ComponentProps<"div">) {
  return (
    <div
      data-slot="settings-row-content"
      className={cn(useLayoutClass("content"), className)}
      {...props}
    />
  );
}

SettingsRowContent.displayName = "SettingsRowContent";

function SettingsRowSkeleton({
  descriptionLines = 1,
  control,
}: {
  descriptionLines?: 0 | 1 | 2;
  control?: React.ReactNode;
}) {
  return (
    <SettingsRow>
      <SettingsRowHeader>
        <SettingsRowTitle>
          <div className="flex h-5 items-center">
            <Skeleton className="h-3.5 w-24" />
          </div>
        </SettingsRowTitle>
        {descriptionLines > 0 ? (
          <SettingsRowDescription>
            <div className="flex h-5 items-center">
              <Skeleton className="h-3.5 w-full" />
            </div>
            {descriptionLines === 2 ? (
              <div className="flex h-5 items-center">
                <Skeleton className="h-3.5 w-2/3" />
              </div>
            ) : null}
          </SettingsRowDescription>
        ) : null}
      </SettingsRowHeader>
      <SettingsRowContent>
        {control ?? <Skeleton className="h-9 w-full max-w-(--setting-w) rounded-md" />}
      </SettingsRowContent>
    </SettingsRow>
  );
}

SettingsRowSkeleton.displayName = "SettingsRowSkeleton";

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
  SettingsPanel,
  SettingsRow,
  SettingsRowContent,
  SettingsRowDescription,
  SettingsRowHeader,
  SettingsRowSkeleton,
  SettingsRowTitle,
};
