"use client";

import { BrandLogo } from "@/lib/extensions/brand-logos";
import {
  CATEGORY_LABELS,
  EXTENSION_TYPE_LABELS,
  type Extension,
  getExtensionBySlug,
} from "@/lib/extensions/registry";
import { Badge, Button, Tabs, TabsContent, TabsList, TabsTrigger } from "@unkey/ui";
import Link from "next/link";
import { notFound, useParams } from "next/navigation";
import { ProjectContentWrapper } from "../../../components/project-content-wrapper";

export default function ExtensionDetailPage() {
  const params = useParams<{
    workspaceSlug: string;
    projectId: string;
    extensionSlug: string;
  }>();

  const extension = getExtensionBySlug(params.extensionSlug);
  if (!extension) {
    notFound();
  }

  const basePath = `/${params.workspaceSlug}/projects/${params.projectId}/extensions`;

  return (
    <ProjectContentWrapper centered maxWidth="1100px" className="mt-8">
      <Link href={basePath} className="text-[12px] text-grayA-10 hover:text-grayA-12 w-fit">
        ← Back to extensions
      </Link>

      <Header extension={extension} installHref={`${basePath}/${extension.slug}/install`} />

      <div className="grid gap-8 md:grid-cols-[1fr_280px]">
        <DetailTabs extension={extension} />
        <Sidebar extension={extension} />
      </div>
    </ProjectContentWrapper>
  );
}

function Header({
  extension,
  installHref,
}: {
  extension: Extension;
  installHref: string;
}) {
  return (
    <div className="flex flex-col gap-4 md:flex-row md:items-start md:justify-between">
      <div className="flex items-start gap-4">
        <div className="flex h-14 w-14 items-center justify-center rounded-2xl bg-white shadow-sm shadow-grayA-8/30 ring-1 ring-grayA-3 overflow-hidden shrink-0">
          <BrandLogo
            slug={extension.slug}
            iconUrl={extension.iconUrl}
            name={extension.name}
            className="size-9"
          />
        </div>
        <div className="flex flex-col gap-1">
          <div className="flex items-center gap-2">
            <h1 className="text-[20px] font-semibold text-grayA-12">{extension.name}</h1>
            <Badge variant="secondary" size="sm">
              {EXTENSION_TYPE_LABELS[extension.type]}
            </Badge>
            {extension.verified ? (
              <Badge variant="success" size="sm">
                Verified
              </Badge>
            ) : null}
            {extension.scope === "workspace" ? (
              <Badge variant="primary" size="sm">
                Workspace
              </Badge>
            ) : null}
          </div>
          <p className="text-[13px] text-grayA-11">{extension.tagline}</p>
          <p className="text-[12px] text-grayA-10">
            By{" "}
            {extension.vendor.url ? (
              <a
                href={extension.vendor.url}
                target="_blank"
                rel="noreferrer"
                className="hover:text-grayA-12 underline-offset-2 hover:underline"
              >
                {extension.vendor.name}
              </a>
            ) : (
              extension.vendor.name
            )}
          </p>
        </div>
      </div>

      <div className="flex items-center gap-2">
        {extension.links.docs ? (
          <Button variant="outline" asChild>
            <a href={extension.links.docs} target="_blank" rel="noreferrer">
              Docs
            </a>
          </Button>
        ) : null}
        <Button asChild>
          <Link href={installHref}>Install</Link>
        </Button>
      </div>
    </div>
  );
}

function DetailTabs({ extension }: { extension: Extension }) {
  return (
    <Tabs defaultValue="overview">
      <TabsList>
        <TabsTrigger value="overview">Overview</TabsTrigger>
        <TabsTrigger value="configuration" disabled>
          Configuration
        </TabsTrigger>
        <TabsTrigger value="changelog">Changelog</TabsTrigger>
        <TabsTrigger value="permissions">Permissions</TabsTrigger>
      </TabsList>

      <TabsContent value="overview" className="pt-4">
        <p className="text-[13px] leading-relaxed text-grayA-12 whitespace-pre-line">
          {extension.description}
        </p>
      </TabsContent>

      <TabsContent value="changelog" className="pt-4">
        <ul className="flex flex-col gap-4">
          {extension.changelog.map((entry) => (
            <li key={entry.version} className="flex flex-col gap-1 border-l-2 border-grayA-3 pl-4">
              <div className="flex items-center gap-2">
                <span className="text-[13px] font-medium text-grayA-12">{entry.version}</span>
                <span className="text-[11px] text-grayA-10">{entry.date}</span>
              </div>
              <p className="text-[13px] text-grayA-11">{entry.notes}</p>
            </li>
          ))}
        </ul>
      </TabsContent>

      <TabsContent value="permissions" className="pt-4">
        <ul className="flex flex-col gap-3">
          {extension.permissions.map((permission) => (
            <li
              key={permission.id}
              className="flex flex-col gap-1 rounded-md border border-grayA-3 p-3"
            >
              <span className="text-[13px] font-medium text-grayA-12">{permission.label}</span>
              <span className="text-[12px] text-grayA-11">{permission.description}</span>
            </li>
          ))}
        </ul>
      </TabsContent>
    </Tabs>
  );
}

function Sidebar({ extension }: { extension: Extension }) {
  return (
    <aside className="flex flex-col gap-5 rounded-lg border border-grayA-3 p-4 h-fit">
      <SidebarRow label="Installs" value={extension.installs.toLocaleString()} />
      <SidebarRow label="Scope" value={extension.scope === "workspace" ? "Workspace" : "Project"} />
      <SidebarRow label="Last updated" value={extension.changelog.at(0)?.date ?? "—"} />

      <SidebarSection label="Categories">
        <div className="flex flex-wrap gap-1.5">
          {extension.categories.map((category) => (
            <Badge key={category} variant="secondary" size="sm">
              {CATEGORY_LABELS[category]}
            </Badge>
          ))}
        </div>
      </SidebarSection>

      <SidebarSection label="What it accesses">
        <ul className="flex flex-col gap-1">
          {extension.permissions.map((permission) => (
            <li key={permission.id} className="text-[12px] text-grayA-11">
              · {permission.label}
            </li>
          ))}
        </ul>
      </SidebarSection>

      {extension.vendor.supportEmail ? (
        <SidebarRow label="Support" value={extension.vendor.supportEmail} />
      ) : null}
    </aside>
  );
}

function SidebarRow({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex items-center justify-between">
      <span className="text-[11px] uppercase tracking-wide text-grayA-10">{label}</span>
      <span className="text-[12px] text-grayA-12">{value}</span>
    </div>
  );
}

function SidebarSection({
  label,
  children,
}: {
  label: string;
  children: React.ReactNode;
}) {
  return (
    <div className="flex flex-col gap-2">
      <span className="text-[11px] uppercase tracking-wide text-grayA-10">{label}</span>
      {children}
    </div>
  );
}
