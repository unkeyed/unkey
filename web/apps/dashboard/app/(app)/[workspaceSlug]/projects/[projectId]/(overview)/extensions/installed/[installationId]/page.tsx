"use client";

import { Switch } from "@/components/ui/switch";
import { BrandLogo } from "@/lib/extensions/brand-logos";
import { type Installation, useInstallations } from "@/lib/extensions/installations";
import {
  type Extension,
  type ExtensionConfigState,
  getExtensionBySlug,
} from "@/lib/extensions/registry";
import {
  Button,
  CopyButton,
  SettingCard,
  SettingCardGroup,
  SettingsDangerZone,
  Tabs,
  TabsContent,
  TabsList,
  TabsTrigger,
  toast,
} from "@unkey/ui";
import Link from "next/link";
import { notFound, useParams, useRouter } from "next/navigation";
import { useState } from "react";
import { ProjectContentWrapper } from "../../../../components/project-content-wrapper";
import { ConfigForm } from "../../components/config-form";
import { PreviewPill } from "../../components/preview-pill";
import { StatusPill } from "../page";

export default function InstallationDetailPage() {
  const params = useParams<{
    workspaceSlug: string;
    projectId: string;
    installationId: string;
  }>();
  const router = useRouter();

  const basePath = `/${params.workspaceSlug}/projects/${params.projectId}/extensions`;
  const installedPath = `${basePath}/installed`;

  const { installations, update, remove, setEnabled } = useInstallations(params.projectId);
  // Track an in-flight uninstall so the page doesn't 404-flash during the
  // tiny window where the install row has been removed from the store but
  // the redirect to /installed hasn't completed yet.
  const [uninstalling, setUninstalling] = useState(false);
  const installation = installations.find((i) => i.id === params.installationId);
  if (!installation) {
    if (uninstalling) {
      return null;
    }
    notFound();
  }
  const extension = getExtensionBySlug(installation.extensionSlug);
  if (!extension) {
    notFound();
  }

  return (
    <ProjectContentWrapper centered maxWidth="900px" className="my-10 flex flex-col gap-8">
      <Link
        href={installedPath}
        className="text-[12px] text-grayA-10 hover:text-grayA-12 w-fit -mb-2"
      >
        ← Back to installed
      </Link>

      <Header extension={extension} installation={installation} />

      <Tabs defaultValue="overview" className="flex flex-col gap-6">
        <TabsList>
          <TabsTrigger value="overview">Overview</TabsTrigger>
          <TabsTrigger value="configuration">Configuration</TabsTrigger>
          <TabsTrigger value="danger">Danger zone</TabsTrigger>
        </TabsList>

        <TabsContent value="overview" className="flex flex-col gap-6">
          <OverviewTab
            installation={installation}
            extension={extension}
            onSetEnabled={async (enabled) => {
              try {
                await setEnabled(installation.id, enabled);
                toast.success(enabled ? "Extension enabled" : "Extension disabled");
              } catch (err) {
                toast.error(err instanceof Error ? err.message : "Failed to update status");
              }
            }}
          />
        </TabsContent>

        <TabsContent value="configuration" className="flex flex-col gap-6">
          <ConfigurationTab
            extension={extension}
            installation={installation}
            onSave={async (patch) => {
              try {
                await update(installation.id, patch);
                toast.success("Configuration saved");
              } catch (err) {
                toast.error(err instanceof Error ? err.message : "Failed to save configuration");
              }
            }}
          />
        </TabsContent>

        <TabsContent value="danger" className="flex flex-col gap-6">
          <DangerTab
            installation={installation}
            extension={extension}
            onUninstall={async () => {
              setUninstalling(true);
              // Push the route change before awaiting remove(): the live
              // path's tRPC mutation can take a few hundred ms, and the
              // user shouldn't sit on a "vanishing" detail page in the
              // meantime. The guard above prevents notFound() from firing
              // if the store updates before the navigation lands.
              router.push(installedPath);
              try {
                await remove(installation.id);
                toast.success(`Uninstalled ${installation.instanceName}`);
              } catch (err) {
                setUninstalling(false);
                toast.error(err instanceof Error ? err.message : "Failed to uninstall");
              }
            }}
          />
        </TabsContent>
      </Tabs>
    </ProjectContentWrapper>
  );
}

function Header({
  extension,
  installation,
}: {
  extension: Extension;
  installation: Installation;
}) {
  return (
    <div className="flex items-start gap-5 rounded-2xl border border-grayA-3 bg-grayA-1 p-5">
      <div className="flex h-14 w-14 items-center justify-center rounded-2xl bg-white shadow-sm shadow-grayA-8/30 ring-1 ring-grayA-3 overflow-hidden shrink-0">
        <BrandLogo
          slug={extension.slug}
          iconUrl={extension.iconUrl}
          name={extension.name}
          className="size-8"
        />
      </div>
      <div className="flex flex-col gap-1.5 flex-1 min-w-0">
        <div className="flex items-center gap-2 flex-wrap">
          <h1 className="text-[18px] font-semibold text-grayA-12 truncate">
            {installation.instanceName}
          </h1>
          <StatusPill status={installation.status} />
          <PreviewPill extension={extension} />
        </div>
        <p className="text-[12px] text-grayA-10">{extension.tagline}</p>
        <div className="flex items-center gap-3 text-[11px] text-grayA-9 mt-1">
          <span>{extension.name}</span>
          <span className="text-grayA-6">·</span>
          <span>Installed {formatDate(installation.installedAt)}</span>
          <span className="text-grayA-6">·</span>
          <span className="font-mono">{installation.id}</span>
          <CopyButton value={installation.id} />
        </div>
      </div>
    </div>
  );
}

function OverviewTab({
  installation,
  extension,
  onSetEnabled,
}: {
  installation: Installation;
  extension: Extension;
  onSetEnabled: (enabled: boolean) => void | Promise<void>;
}) {
  const enabled = installation.status !== "disabled";
  return (
    <div className="flex flex-col gap-6">
      <SettingCardGroup>
        <SettingCard
          title={enabled ? "Enabled" : "Disabled"}
          description={
            enabled
              ? "Extension is running and processing events for this project."
              : "Paused. Configuration is preserved — re-enable to resume."
          }
          border="both"
        >
          <Switch checked={enabled} onCheckedChange={(checked) => onSetEnabled(checked === true)} />
        </SettingCard>
      </SettingCardGroup>

      <SettingCardGroup>
        <SettingCard title="Extension" description="The marketplace entry backing this install.">
          <span className="text-[13px] text-grayA-12">{extension.name}</span>
        </SettingCard>
        <SettingCard title="Instance name" description="How this install is labelled in lists.">
          <span className="text-[13px] text-grayA-12">{installation.instanceName}</span>
        </SettingCard>
        <SettingCard title="Scope" description="Whether the install is workspace- or project-wide.">
          <span className="text-[13px] text-grayA-12">
            {extension.scope === "workspace" ? "Workspace" : "Project"}
          </span>
        </SettingCard>
        <SettingCard title="OAuth" description="Provider connection state, when applicable.">
          <span className="text-[13px] text-grayA-12">
            {extension.oauth ? (installation.oauthConnected ? "Connected" : "Not connected") : "—"}
          </span>
        </SettingCard>
        <SettingCard
          title="Last event"
          description="Most recent event the provider observed for this install."
        >
          <span className="text-[13px] text-grayA-12">
            {installation.lastEventAt ? formatDate(installation.lastEventAt) : "Never"}
          </span>
        </SettingCard>
      </SettingCardGroup>
    </div>
  );
}

function ConfigurationTab({
  extension,
  installation,
  onSave,
}: {
  extension: Extension;
  installation: Installation;
  onSave: (patch: Partial<Installation>) => void;
}) {
  const [values, setValues] = useState<ExtensionConfigState>(installation.config);
  const isDirty = JSON.stringify(values) !== JSON.stringify(installation.config);

  return (
    <div className="flex flex-col gap-4 rounded-[14px] border border-grayA-4 bg-grayA-1 p-6">
      <div className="flex flex-col gap-1">
        <h2 className="text-[14px] font-semibold text-grayA-12">Configuration</h2>
        <p className="text-[12px] text-grayA-10">
          Edit the values you supplied at install time. Saving applies them immediately.
        </p>
      </div>
      <ConfigForm fields={extension.configFields} values={values} onChange={setValues} />
      <div className="flex justify-end pt-2">
        <Button onClick={() => onSave({ config: values })} disabled={!isDirty}>
          Save changes
        </Button>
      </div>
    </div>
  );
}

function DangerTab({
  installation,
  extension,
  onUninstall,
}: {
  installation: Installation;
  extension: Extension;
  onUninstall: () => void;
}) {
  return (
    <SettingsDangerZone>
      <UninstallRow installation={installation} extension={extension} onConfirm={onUninstall} />
    </SettingsDangerZone>
  );
}

function UninstallRow({
  installation,
  extension,
  onConfirm,
}: {
  installation: Installation;
  extension: Extension;
  onConfirm: () => void;
}) {
  const [confirming, setConfirming] = useState(false);
  return (
    <div className="flex items-center justify-between p-4 gap-4">
      <div className="space-y-1 min-w-0">
        <p className="font-medium text-gray-12 text-sm">Uninstall extension</p>
        <p className="text-gray-11 text-[13px]">
          Removes "{installation.instanceName}" from this project. {extension.name} will stop
          receiving events immediately. This cannot be undone.
        </p>
      </div>
      {confirming ? (
        <div className="flex items-center gap-2 shrink-0">
          <Button variant="outline" onClick={() => setConfirming(false)}>
            Cancel
          </Button>
          <Button
            variant="destructive"
            onClick={() => {
              setConfirming(false);
              onConfirm();
            }}
          >
            Confirm uninstall
          </Button>
        </div>
      ) : (
        <Button variant="destructive" className="shrink-0" onClick={() => setConfirming(true)}>
          Uninstall
        </Button>
      )}
    </div>
  );
}

function formatDate(iso: string): string {
  const d = new Date(iso);
  return Number.isNaN(d.getTime())
    ? iso
    : d.toLocaleDateString(undefined, {
        year: "numeric",
        month: "short",
        day: "numeric",
        hour: "numeric",
        minute: "2-digit",
      });
}
