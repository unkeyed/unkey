"use client";

import { AddEnvVarsPanel } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/apps/_components/env-vars/add/add-env-vars-button";
import { useAppEnvVars } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/apps/_components/env-vars/use-app-env-vars";
import { BuildCommand } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/apps/_components/settings/build-settings/build-command-settings";
import { Dockerfile } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/apps/_components/settings/build-settings/dockerfile-settings";
import { RootDirectory } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/apps/_components/settings/build-settings/root-directory-settings";
import { WatchPaths } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/apps/_components/settings/build-settings/watch-paths-settings";
import {
  formatInstances,
  readCompute,
} from "@/app/(app)/[workspaceSlug]/projects/[projectId]/apps/_components/settings/compute/draft";
import { useEnvironmentSettings } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/apps/_components/settings/environment-provider";
import { useBuildRecipe } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/apps/_components/settings/hooks/use-build-recipe";
import { useBuildSource } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/apps/_components/settings/hooks/use-build-source";
import { OCIImage } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/apps/_components/settings/oci-image";
import {
  CommandForm,
  ComputeForm,
} from "@/app/(app)/[workspaceSlug]/projects/[projectId]/apps/_components/settings/runtime/runtime-form";
import type { EnvironmentSettings } from "@/lib/collections/deploy/environment-settings";
import { plural } from "@/lib/fmt";
import {
  IconHammer2Outline18,
  IconKey2Outline18,
  IconLayers3Outline18,
  type IconProps,
} from "@unkey/icons";
import { match } from "@unkey/match";
import { Button, SettingsForm, SettingsPanel, Skeleton } from "@unkey/ui";
import { type ComponentType, type ReactNode, useEffect, useState } from "react";
import type { SourceKind } from "../wizard-model";

type BlockId = "build" | "scaling" | "env";

type PanelProps = {
  source: SourceKind;
  isOpen: boolean;
  onClose: () => void;
  onSaved: () => void;
};

const BLOCK_ORDER: readonly BlockId[] = ["build", "scaling", "env"];

const BLOCKS: Record<
  BlockId,
  { title: string; icon: ComponentType<IconProps>; Panel: ComponentType<PanelProps> }
> = {
  build: { title: "Build", icon: IconHammer2Outline18, Panel: BuildPanel },
  scaling: { title: "Scaling", icon: IconLayers3Outline18, Panel: ScalingPanel },
  env: { title: "Environment variables", icon: IconKey2Outline18, Panel: EnvVarsPanel },
};

const FLASH_MS = 1200;

function rootLabel(dockerContext: string): string {
  return dockerContext === "." || dockerContext === "" ? "./" : dockerContext;
}

const BUILD_SUMMARY: Record<
  SourceKind,
  (input: { settings: EnvironmentSettings; imageReference: string | null }) => string
> = {
  git: ({ settings }) =>
    settings.dockerfile
      ? `${settings.dockerfile} at ${rootLabel(settings.dockerContext)}`
      : `Automatic build at ${rootLabel(settings.dockerContext)}`,
  oci: ({ imageReference }) => imageReference || "No image yet",
};

function useBlockSummaries(source: SourceKind): Record<BlockId, string> {
  const { settings } = useEnvironmentSettings();
  const { projectId, appId, app } = useBuildSource();
  const envVars = useAppEnvVars(projectId, appId);
  const compute = readCompute(settings);
  return {
    build: BUILD_SUMMARY[source]({ settings, imageReference: app?.imageReference ?? null }),
    scaling: `${formatInstances(compute.replicas)} in ${plural(compute.regions.length, "region")}`,
    env: match(envVars)
      .with({ status: "loading" }, () => "Loading…")
      .with({ status: "failed" }, () => "Could not load variables")
      .with({ status: "ready" }, ({ data }) => {
        const count = new Set(data.map((v) => v.key)).size;
        return count === 0 ? "None yet" : plural(count, "variable");
      })
      .exhaustive(),
  };
}

function useFlash() {
  const [flashed, setFlashed] = useState<BlockId | null>(null);
  useEffect(() => {
    if (flashed === null) {
      return;
    }
    const timer = setTimeout(() => setFlashed(null), FLASH_MS);
    return () => clearTimeout(timer);
  }, [flashed]);
  return { flashed, flash: setFlashed };
}

export function ConfigBlocks({ source }: { source: SourceKind }) {
  const summaries = useBlockSummaries(source);
  const [open, setOpen] = useState<BlockId | null>(null);
  const { flashed, flash } = useFlash();
  const close = () => setOpen(null);

  return (
    <BlocksSection>
      {BLOCK_ORDER.map((id) => {
        const { Panel } = BLOCKS[id];
        return (
          <div key={id}>
            <BlockRow
              id={id}
              hint={summaries[id]}
              flashed={flashed === id}
              onClick={() => setOpen(id)}
            />
            <Panel
              source={source}
              isOpen={open === id}
              onClose={close}
              onSaved={() => {
                close();
                flash(id);
              }}
            />
          </div>
        );
      })}
    </BlocksSection>
  );
}

export function ConfigBlocksSkeleton() {
  return (
    <BlocksSection>
      {BLOCK_ORDER.map((id) => (
        <BlockRow
          key={id}
          id={id}
          hint={<Skeleton className="inline-block h-3 w-32 align-middle" />}
        />
      ))}
    </BlocksSection>
  );
}

function BlocksSection({ children }: { children: ReactNode }) {
  return (
    <section className="flex flex-col gap-3">
      <h2 className="text-sm font-medium text-gray-12">Configuration</h2>
      {children}
    </section>
  );
}

function BlockRow({
  id,
  hint,
  flashed = false,
  onClick,
}: {
  id: BlockId;
  hint: ReactNode;
  flashed?: boolean;
  onClick?: () => void;
}) {
  const { title, icon: Icon } = BLOCKS[id];
  return (
    <button
      type="button"
      disabled={onClick === undefined}
      data-flash={flashed}
      onClick={onClick}
      className="flex w-full items-center gap-3 rounded-lg border bg-raised px-3 py-3 text-left transition-colors duration-500 enabled:cursor-pointer enabled:hover:bg-grayA-2 focus-visible:outline-hidden focus-visible:ring-3 focus-visible:ring-gray-5 data-[flash=true]:border-grayA-6 data-[flash=true]:bg-grayA-2"
    >
      <span className="flex size-8 shrink-0 items-center justify-center rounded-md bg-grayA-3 text-gray-11">
        <Icon className="size-4" />
      </span>
      <span className="flex min-w-0 flex-1 flex-col">
        <span className="text-sm font-medium text-gray-12">{title}</span>
        <span className="truncate text-xs text-gray-11">{hint}</span>
      </span>
      <Button render={<span />} variant="outline" size="sm">
        Edit
      </Button>
    </button>
  );
}

function GitBuildFields() {
  const { form, formProps } = useBuildRecipe();
  return (
    <>
      <SettingsForm {...formProps}>
        <div className="grid grid-cols-1 gap-5 @lg:grid-cols-2 @lg:gap-4">
          <RootDirectory form={form} />
          <Dockerfile form={form} />
        </div>
        <BuildCommand form={form} />
      </SettingsForm>
      <CommandForm />
      <WatchPaths />
    </>
  );
}

function ImageBuildFields() {
  const { app } = useBuildSource();
  return (
    <>
      {app ? <OCIImage appId={app.id} imageReference={app.imageReference ?? ""} /> : null}
      <CommandForm />
    </>
  );
}

const BUILD_FIELDS: Record<SourceKind, ComponentType> = {
  git: GitBuildFields,
  oci: ImageBuildFields,
};

function BuildPanel({ source, ...panel }: PanelProps) {
  const Fields = BUILD_FIELDS[source];
  return (
    <SettingsPanel {...panel} title="Build" description="How Unkey builds your app.">
      <Fields />
    </SettingsPanel>
  );
}

function ScalingPanel({ source: _, ...panel }: PanelProps) {
  return (
    <SettingsPanel
      {...panel}
      title="Scaling"
      description="Where your app runs and how far it scales."
    >
      <ComputeForm hideLockedSizes />
    </SettingsPanel>
  );
}

function EnvVarsPanel({ isOpen, onClose, onSaved }: PanelProps) {
  return <AddEnvVarsPanel mode="manage" isOpen={isOpen} onClose={onClose} onSaved={onSaved} />;
}
