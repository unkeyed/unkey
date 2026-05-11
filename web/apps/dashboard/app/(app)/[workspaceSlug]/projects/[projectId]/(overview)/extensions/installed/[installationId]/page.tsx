"use client";

import { Switch } from "@/components/ui/switch";
import { BrandLogo } from "@/lib/extensions/brand-logos";
import { type Installation, useInstallations } from "@/lib/extensions/installations";
import {
  type Extension,
  type ExtensionConfigState,
  getExtensionBySlug,
} from "@/lib/extensions/registry";
import { Button, ConfirmPopover, Tabs, TabsContent, TabsList, TabsTrigger, toast } from "@unkey/ui";
import Link from "next/link";
import { notFound, useParams, useRouter } from "next/navigation";
import { useRef, useState } from "react";
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
    <ProjectContentWrapper centered maxWidth="900px" className="mt-8">
      <Link href={installedPath} className="text-[12px] text-grayA-10 hover:text-grayA-12 w-fit">
        ← Back to installed
      </Link>

      <Header extension={extension} installation={installation} />

      <Tabs defaultValue="overview">
        <TabsList>
          <TabsTrigger value="overview">Overview</TabsTrigger>
          <TabsTrigger value="configuration">Configuration</TabsTrigger>
          <TabsTrigger value="danger">Danger zone</TabsTrigger>
        </TabsList>

        <TabsContent value="overview" className="pt-4">
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

        <TabsContent value="configuration" className="pt-4">
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

        <TabsContent value="danger" className="pt-4">
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
    <div className="flex items-center gap-4">
      <div className="flex h-12 w-12 items-center justify-center rounded-2xl bg-white shadow-sm shadow-grayA-8/30 ring-1 ring-grayA-3 overflow-hidden shrink-0">
        <BrandLogo
          slug={extension.slug}
          iconUrl={extension.iconUrl}
          name={extension.name}
          className="size-7"
        />
      </div>
      <div className="flex flex-col gap-1">
        <div className="flex items-center gap-2">
          <h1 className="text-[18px] font-semibold text-grayA-12">{installation.instanceName}</h1>
          <StatusPill status={installation.status} />
          <PreviewPill extension={extension} />
        </div>
        <p className="text-[12px] text-grayA-10">
          {extension.name} · installed {new Date(installation.installedAt).toLocaleString()}
        </p>
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
    <div className="flex flex-col gap-4">
      <div className="flex items-center justify-between rounded-lg border border-grayA-3 bg-grayA-1 px-4 py-3">
        <div className="flex flex-col">
          <span className="text-[13px] font-medium text-grayA-12">
            {enabled ? "Enabled" : "Disabled"}
          </span>
          <span className="text-[12px] text-grayA-10">
            {enabled
              ? "Extension is running and processing events for this project."
              : "Paused. Configuration is preserved — re-enable to resume."}
          </span>
        </div>
        <Switch checked={enabled} onCheckedChange={(checked) => onSetEnabled(checked === true)} />
      </div>
      <Row label="Extension" value={extension.name} />
      <Row label="Instance name" value={installation.instanceName} />
      <Row label="Status" value={<StatusPill status={installation.status} />} />
      <Row label="Scope" value={extension.scope === "workspace" ? "Workspace" : "Project"} />
      <Row
        label="OAuth"
        value={
          extension.oauth ? (installation.oauthConnected ? "Connected" : "Not connected") : "—"
        }
      />
      <Row
        label="Last event"
        value={installation.lastEventAt ? new Date(installation.lastEventAt).toLocaleString() : "—"}
      />
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

  const handleSave = () => {
    onSave({ config: values });
  };

  return (
    <div className="flex flex-col gap-4">
      <ConfigForm fields={extension.configFields} values={values} onChange={setValues} />

      <div className="flex justify-end">
        <Button onClick={handleSave} disabled={!isDirty}>
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
  const [open, setOpen] = useState(false);
  const triggerRef = useRef<HTMLButtonElement>(null);

  return (
    <div className="flex flex-col gap-3 rounded-lg border border-errorA-4 p-4">
      <div className="flex flex-col gap-1">
        <h3 className="text-[14px] font-semibold text-grayA-12">Uninstall extension</h3>
        <p className="text-[12px] text-grayA-10">
          Remove "{installation.instanceName}" from this project. {extension.name} will stop
          receiving events immediately. This action cannot be undone.
        </p>
      </div>
      <div>
        <Button
          ref={triggerRef}
          variant="outline"
          color="danger"
          onClick={() => setOpen(true)}
          className="w-fit"
        >
          Uninstall
        </Button>
        <ConfirmPopover
          isOpen={open}
          onOpenChange={setOpen}
          onConfirm={() => {
            setOpen(false);
            onUninstall();
          }}
          triggerRef={triggerRef}
          title={`Uninstall ${installation.instanceName}?`}
          description="This will stop event delivery and delete the configuration."
          confirmButtonText="Uninstall"
          variant="danger"
        />
      </div>
    </div>
  );
}

function Row({ label, value }: { label: string; value: React.ReactNode }) {
  return (
    <div className="flex items-center justify-between border-b border-grayA-3 pb-3 last:border-b-0">
      <span className="text-[12px] uppercase tracking-wide text-grayA-10">{label}</span>
      <span className="text-[13px] text-grayA-12">{value}</span>
    </div>
  );
}
