"use client";

import { BrandLogo } from "@/lib/extensions/brand-logos";
import { type Installation, useInstallations } from "@/lib/extensions/installations";
import { type Extension, getExtensionBySlug } from "@/lib/extensions/registry";
import { Badge, Button } from "@unkey/ui";
import { cn } from "@unkey/ui/src/lib/utils";
import Link from "next/link";
import { useParams } from "next/navigation";
import { ProjectContentWrapper } from "../../../components/project-content-wrapper";
import { ExtensionsHeader } from "../components/extensions-header";
import { PreviewPill } from "../components/preview-pill";

export default function InstalledExtensionsPage() {
  const params = useParams<{ workspaceSlug: string; projectId: string }>();
  const basePath = `/${params.workspaceSlug}/projects/${params.projectId}/extensions`;
  const { installations } = useInstallations(params.projectId);

  return (
    <ProjectContentWrapper centered maxWidth="1200px" className="mt-8">
      <ExtensionsHeader
        basePath={basePath}
        active="installed"
        installedCount={installations.length}
      />

      {installations.length === 0 ? (
        <EmptyInstalled basePath={basePath} />
      ) : (
        <InstalledTable installations={installations} basePath={basePath} />
      )}
    </ProjectContentWrapper>
  );
}

function InstalledTable({
  installations,
  basePath,
}: {
  installations: Installation[];
  basePath: string;
}) {
  return (
    <div className="overflow-hidden rounded-lg border border-grayA-3">
      <table className="w-full text-[13px]">
        <thead className="bg-grayA-2 text-[11px] uppercase tracking-wide text-grayA-10">
          <tr>
            <th className="px-4 py-2.5 text-left font-medium">Extension</th>
            <th className="px-4 py-2.5 text-left font-medium">Status</th>
            <th className="px-4 py-2.5 text-left font-medium">Installed</th>
            <th className="px-4 py-2.5 text-right font-medium">Actions</th>
          </tr>
        </thead>
        <tbody>
          {installations.map((installation) => {
            const extension = getExtensionBySlug(installation.extensionSlug);
            const detailHref = `${basePath}/installed/${installation.id}`;
            return (
              <tr
                key={installation.id}
                className="border-t border-grayA-3 hover:bg-grayA-2 transition-colors"
              >
                <td className="px-4 py-3">
                  <Link href={detailHref} className="flex items-center gap-3">
                    {extension ? <ExtensionLogo extension={extension} /> : <FallbackLogo />}
                    <div className="flex flex-col">
                      <span className="font-medium text-grayA-12">
                        {extension?.name ?? installation.extensionSlug}
                      </span>
                      <span className="text-[12px] text-grayA-10">{installation.instanceName}</span>
                    </div>
                  </Link>
                </td>
                <td className="px-4 py-3">
                  <div className="flex items-center gap-1.5">
                    <StatusPill status={installation.status} />
                    {extension ? <PreviewPill extension={extension} /> : null}
                  </div>
                </td>
                <td className="px-4 py-3 text-grayA-11">
                  {formatRelative(installation.installedAt)}
                </td>
                <td className="px-4 py-3 text-right">
                  <Button variant="outline" size="sm" asChild>
                    <Link href={detailHref}>Manage</Link>
                  </Button>
                </td>
              </tr>
            );
          })}
        </tbody>
      </table>
    </div>
  );
}

function EmptyInstalled({ basePath }: { basePath: string }) {
  return (
    <div className="flex flex-col items-center justify-center gap-3 rounded-lg border border-dashed border-grayA-4 py-16 text-center">
      <p className="text-[14px] font-medium text-grayA-12">No extensions installed yet</p>
      <p className="text-[12px] text-grayA-10 max-w-sm">
        Browse the marketplace to add logging, alerting, or analytics integrations to this project.
      </p>
      <Button asChild>
        <Link href={basePath}>Browse marketplace</Link>
      </Button>
    </div>
  );
}

function ExtensionLogo({ extension }: { extension: Extension }) {
  return (
    <div className="flex h-9 w-9 items-center justify-center rounded-[10px] bg-white shadow-sm shadow-grayA-8/20 ring-1 ring-grayA-3 overflow-hidden shrink-0">
      <BrandLogo
        slug={extension.slug}
        iconUrl={extension.iconUrl}
        name={extension.name}
        className="size-5"
      />
    </div>
  );
}

function FallbackLogo() {
  return <div className="h-9 w-9 rounded-[10px] bg-grayA-2 ring-1 ring-grayA-3" />;
}

const STATUS_LABELS: Record<Installation["status"], string> = {
  active: "Active",
  degraded: "Degraded",
  disabled: "Disabled",
  verifying: "Verifying",
  failed: "Failed",
};

export function StatusPill({ status }: { status: Installation["status"] }) {
  const variant =
    status === "active"
      ? "success"
      : status === "degraded" || status === "verifying"
        ? "warning"
        : status === "failed"
          ? "error"
          : "secondary";
  return (
    <Badge variant={variant} size="sm" className={cn("font-normal")}>
      {STATUS_LABELS[status]}
    </Badge>
  );
}

function formatRelative(iso: string): string {
  const then = new Date(iso).getTime();
  if (Number.isNaN(then)) {
    return "—";
  }
  const diff = Date.now() - then;
  const minutes = Math.round(diff / 60_000);
  if (minutes < 1) {
    return "just now";
  }
  if (minutes < 60) {
    return `${minutes}m ago`;
  }
  const hours = Math.round(minutes / 60);
  if (hours < 24) {
    return `${hours}h ago`;
  }
  const days = Math.round(hours / 24);
  if (days < 30) {
    return `${days}d ago`;
  }
  return new Date(iso).toLocaleDateString();
}
