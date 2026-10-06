"use client";

import {
  findDockerfiles,
  suggestRootDirectories,
} from "@/app/(app)/[workspaceSlug]/projects/_components/repo-tree";
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from "@/components/ui/collapsible";
import { FormCombobox } from "@/components/ui/form-combobox";
import { collection } from "@/lib/collections";
import {
  type EnvironmentSettings,
  SILENT_SAVE,
} from "@/lib/collections/deploy/environment-settings";
import { getErrorMessage } from "@/lib/unkey-client";
import { zodResolver } from "@hookform/resolvers/zod";
import { IconChevronRightOutline18 } from "@unkey/icons";
import { Button, FormInput, toast } from "@unkey/ui";
import { cn } from "@unkey/ui/src/lib/utils";
import {
  type ReactNode,
  useEffect,
  useId,
  useMemo,
  useRef,
  useState,
  useSyncExternalStore,
} from "react";
import { type FieldErrors, useForm, useWatch } from "react-hook-form";
import { type BackgroundWork, WORK_DONE, type WorkStatus } from "../../background-work";
import { useNewAppFlow } from "../../flow";
import type { SourceKind } from "../../wizard-model";
import { PaneSubmit } from "../pane-actions";
import {
  type BuildMethod,
  type DeploymentConfig,
  type SettingField,
  applyDeploymentConfig,
  deploymentConfigSchema,
  directoryLabel,
  readDeploymentConfig,
  resolveBuildMethod,
  settingTitle,
  settingsLayout,
} from "./deployment-config";
import { RegionSelect } from "./region-select";
import { SizeField } from "./size-field";
import type { RepoTree } from "./use-repo-tree";

type SettingsFormProps = {
  appId: string;
  source: SourceKind;
  production: EnvironmentSettings;
  environmentIds: string[];
  onSaved: () => void | Promise<void>;
  repoTree?: RepoTree;
  pairedField?: { title: string; control: ReactNode };
  advancedLeadingRows?: ReactNode;
  sections?: { title: string; hint?: string; content: ReactNode }[];
  focusField?: SettingField | null;
};

const noRepoTree: RepoTree = { entries: [], loading: false };

const noSubscription = () => () => {};

function useWorkStatus(work: BackgroundWork | null): WorkStatus {
  return useSyncExternalStore(work?.subscribe ?? noSubscription, () => work?.status() ?? WORK_DONE);
}

function loadingOption(label: string) {
  return { label: <span className="text-gray-10">{label}</span>, value: label, disabled: true };
}

const buildCommandField: Record<BuildMethod, { description: string; disabled: boolean }> = {
  automatic: { description: "Leave empty to detect it automatically.", disabled: false },
  dockerfile: { description: "Not used when a Dockerfile is set.", disabled: true },
};

function FolderIcon() {
  return (
    <svg viewBox="0 0 18 18" aria-hidden="true" className="size-3.5 shrink-0 text-gray-11">
      <path
        d="M2.25 4.5A1.5 1.5 0 0 1 3.75 3h3.19a1.5 1.5 0 0 1 1.2.6L9 4.75h5.25a1.5 1.5 0 0 1 1.5 1.5v7.25a1.5 1.5 0 0 1-1.5 1.5H3.75a1.5 1.5 0 0 1-1.5-1.5Z"
        fill="none"
        stroke="currentColor"
        strokeWidth="1.5"
        strokeLinejoin="round"
      />
    </svg>
  );
}

function DirectoryLabel({ path }: { path: string }) {
  return (
    <span className="flex min-w-0 items-center gap-2">
      <FolderIcon />
      <span className="truncate">{directoryLabel(path)}</span>
    </span>
  );
}

const HIGHLIGHT_MS = 2400;

export function Field({
  title,
  highlight = false,
  children,
}: {
  title: ReactNode;
  highlight?: boolean;
  children: ReactNode;
}) {
  const ref = useRef<HTMLDivElement>(null);
  const [lit, setLit] = useState(highlight);
  // biome-ignore lint/correctness/useExhaustiveDependencies: run once when the field mounts highlighted.
  useEffect(() => {
    const el = ref.current;
    if (!highlight || !el) {
      return;
    }
    const reduced = window.matchMedia("(prefers-reduced-motion: reduce)").matches;
    el.scrollIntoView({ block: "center", behavior: reduced ? "auto" : "smooth" });
    el.querySelector<HTMLElement>("button, input")?.focus({ preventScroll: true });
    const timer = setTimeout(() => setLit(false), HIGHLIGHT_MS);
    return () => clearTimeout(timer);
  }, []);
  return (
    <div
      ref={ref}
      className={cn(
        "flex min-w-0 flex-col gap-2 rounded-md outline-offset-4 transition-[outline-color] duration-500 motion-reduce:transition-none",
        lit ? "outline-2 outline-warning-7" : "outline-2 outline-transparent",
      )}
    >
      <div className="text-sm font-medium text-gray-12">{title}</div>
      {children}
    </div>
  );
}

const REVEAL_WINDOW_MS = 500;

function useRevealOnOpen(open: boolean) {
  const ref = useRef<HTMLDivElement>(null);
  useEffect(() => {
    const el = ref.current;
    if (!open || !el) {
      return;
    }
    const behavior = window.matchMedia("(prefers-reduced-motion: reduce)").matches
      ? "auto"
      : "smooth";
    const observer = new ResizeObserver(() => el.scrollIntoView({ block: "nearest", behavior }));
    observer.observe(el);
    const stop = setTimeout(() => observer.disconnect(), REVEAL_WINDOW_MS);
    return () => {
      observer.disconnect();
      clearTimeout(stop);
    };
  }, [open]);
  return ref;
}

function Disclosure({
  title,
  hint,
  open: controlledOpen,
  onOpenChange,
  children,
}: {
  title: string;
  hint?: string;
  open?: boolean;
  onOpenChange?: (open: boolean) => void;
  children: ReactNode;
}) {
  const [ownOpen, setOwnOpen] = useState(false);
  const open = controlledOpen ?? ownOpen;
  const setOpen = onOpenChange ?? setOwnOpen;
  const revealRef = useRevealOnOpen(open);
  return (
    <Collapsible ref={revealRef} className="scroll-my-5" open={open} onOpenChange={setOpen}>
      <CollapsibleTrigger className="flex w-full items-center gap-2 rounded-md px-1 py-1.5 text-sm font-medium text-gray-12">
        <IconChevronRightOutline18
          className={cn(
            "size-3 text-gray-9 transition-transform duration-200",
            open && "rotate-90",
          )}
        />
        {title}
        {hint ? <span className="ml-auto text-xs font-normal text-gray-9">{hint}</span> : null}
      </CollapsibleTrigger>
      <CollapsibleContent className="pt-2">{children}</CollapsibleContent>
    </Collapsible>
  );
}

export function FieldStack({ children }: { children: ReactNode }) {
  return (
    <div className="@container flex flex-col gap-5 rounded-lg border border-grayA-4 bg-raised p-5">
      {children}
    </div>
  );
}

export function SettingsForm({
  appId,
  source,
  production,
  environmentIds,
  onSaved,
  repoTree = noRepoTree,
  pairedField,
  advancedLeadingRows,
  sections = [],
  focusField = null,
}: SettingsFormProps) {
  const layout = settingsLayout[source];
  const [advancedOpen, setAdvancedOpen] = useState(
    focusField !== null && layout.advanced.includes(focusField),
  );
  const { setup } = useNewAppFlow();
  const work = setup?.appId === appId ? setup.work : null;
  const workStatus = useWorkStatus(work);
  const tree = repoTree.entries;

  const {
    register,
    handleSubmit,
    setValue,
    control,
    formState: { errors, isSubmitting },
  } = useForm<DeploymentConfig>({
    resolver: zodResolver(deploymentConfigSchema),
    mode: "onChange",
    defaultValues: readDeploymentConfig(production),
  });

  const dockerContext = useWatch({ control, name: "dockerContext" });
  const dockerfile = useWatch({ control, name: "dockerfile" });
  const regions = useWatch({ control, name: "regions" });
  const size = useWatch({ control, name: "size" });
  const buildCommand = buildCommandField[resolveBuildMethod(dockerfile)];

  const rootDirectoryOptions = useMemo(
    () => [
      ...suggestRootDirectories(tree).map(({ path, marker }) => ({
        label: (
          <span className="flex min-w-0 items-center gap-2">
            <DirectoryLabel path={path} />
            <span className="truncate text-xs text-gray-9">{marker}</span>
          </span>
        ),
        selectedLabel: <DirectoryLabel path={path} />,
        value: path,
        searchValue: path === "." ? ". repository root" : path,
      })),
      ...(repoTree.loading ? [loadingOption("Loading directories…")] : []),
    ],
    [tree, repoTree.loading],
  );

  const dockerfileOptions = useMemo(
    () => [
      {
        label: <span className="text-gray-11">Automatic (no Dockerfile)</span>,
        selectedLabel: "Automatic (no Dockerfile)",
        value: "",
        searchValue: "automatic (no dockerfile)",
      },
      ...findDockerfiles(tree, dockerContext).map((path) => ({
        label: path,
        selectedLabel: path,
        value: path,
        searchValue: path,
      })),
      ...(repoTree.loading ? [loadingOption("Loading Dockerfiles…")] : []),
    ],
    [tree, dockerContext, repoTree.loading],
  );

  const controls: Record<SettingField, ReactNode> = {
    dockerContext: (
      <FormCombobox
        aria-label="Root directory"
        error={errors.dockerContext?.message}
        options={rootDirectoryOptions}
        value={dockerContext}
        onSelect={(value) => setValue("dockerContext", value, { shouldValidate: true })}
        creatable
        searchPlaceholder="Search or enter a path…"
        emptyMessage={
          <div className="mt-2">
            {repoTree.loading ? "Loading directories…" : "No app directories detected"}
          </div>
        }
        placeholder={<DirectoryLabel path="." />}
      />
    ),
    regions: (
      <RegionSelect
        regions={regions}
        error={errors.regions?.message}
        onChange={(next) => setValue("regions", next, { shouldValidate: true })}
      />
    ),
    port: (
      <FormInput
        aria-label="Port"
        data-1p-ignore
        autoComplete="off"
        type="number"
        inputMode="numeric"
        error={errors.port?.message}
        {...register("port", { valueAsNumber: true })}
      />
    ),
    dockerfile: (
      <FormCombobox
        aria-label="Dockerfile"
        error={errors.dockerfile?.message}
        options={dockerfileOptions}
        value={dockerfile}
        onSelect={(value) => setValue("dockerfile", value, { shouldValidate: true })}
        creatable
        searchPlaceholder="Search or enter a path…"
        emptyMessage={
          <div className="mt-2">
            {repoTree.loading ? "Loading Dockerfiles…" : "No Dockerfiles detected"}
          </div>
        }
        placeholder={<span className="text-grayA-8">Automatic (no Dockerfile)</span>}
      />
    ),
    buildCommand: (
      <FormInput
        aria-label="Build command"
        data-1p-ignore
        autoComplete="off"
        description={buildCommand.description}
        placeholder="Automatic"
        disabled={buildCommand.disabled}
        error={errors.buildCommand?.message}
        {...register("buildCommand")}
      />
    ),
    startCommand: (
      <FormInput
        aria-label="Start command"
        data-1p-ignore
        autoComplete="off"
        description="Leave empty to detect it automatically."
        placeholder="Automatic"
        error={errors.startCommand?.message}
        {...register("startCommand")}
      />
    ),
    size: (
      <SizeField size={size} onChange={(next) => setValue("size", next, { shouldDirty: true })} />
    ),
  };

  const rows = (fields: readonly SettingField[]) =>
    fields.map((field) => (
      <Field key={field} title={settingTitle[field]} highlight={field === focusField}>
        {controls[field]}
      </Field>
    ));

  // The default settings land in the background after the app is created; a
  // save sent earlier could be overwritten by them.
  const onValid = async (config: DeploymentConfig) => {
    try {
      await work?.settled();
    } catch {
      return;
    }
    const transaction = collection.environmentSettings.update(
      environmentIds,
      { metadata: SILENT_SAVE },
      (drafts) => {
        for (const draft of drafts) {
          applyDeploymentConfig(draft, config);
        }
      },
    );
    try {
      await transaction.isPersisted.promise;
      await onSaved();
    } catch (error) {
      toast.error("Could not save the settings", { description: getErrorMessage(error) });
    }
  };

  const onInvalid = (fieldErrors: FieldErrors<DeploymentConfig>) => {
    if (layout.advanced.some((field) => fieldErrors[field])) {
      setAdvancedOpen(true);
    }
  };

  const formId = useId();
  return (
    <form
      id={formId}
      className="flex flex-1 flex-col gap-3"
      onSubmit={handleSubmit(onValid, onInvalid)}
    >
      <FieldStack>
        {pairedField && layout.main[0] === "dockerContext" ? (
          <>
            <div className="grid grid-cols-2 gap-4">
              <Field title={pairedField.title}>{pairedField.control}</Field>
              <Field title={settingTitle.dockerContext}>{controls.dockerContext}</Field>
            </div>
            {rows(layout.main.slice(1))}
          </>
        ) : (
          rows(layout.main)
        )}
      </FieldStack>
      {layout.advanced.length > 0 || advancedLeadingRows ? (
        <Disclosure
          title="Advanced"
          hint="Defaults"
          open={advancedOpen}
          onOpenChange={setAdvancedOpen}
        >
          <FieldStack>
            {advancedLeadingRows}
            {rows(layout.advanced)}
          </FieldStack>
        </Disclosure>
      ) : null}
      {sections.map((section) => (
        <Disclosure key={section.title} title={section.title} hint={section.hint}>
          {section.content}
        </Disclosure>
      ))}
      <PaneSubmit
        form={formId}
        loading={isSubmitting}
        disabled={isSubmitting || workStatus.kind === "failed"}
        notice={
          workStatus.kind === "failed" && work ? (
            <>
              <span className="text-xs text-error-11">{getErrorMessage(workStatus.error)}</span>
              <Button size="sm" variant="outline" onClick={work.retry}>
                Retry
              </Button>
            </>
          ) : null
        }
      />
    </form>
  );
}
