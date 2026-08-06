"use client";

import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { routes } from "@/lib/navigation/routes";
import {
  Button,
  PageBody,
  PageContainer,
  PageHeader,
  PageHeaderActions,
  PageHeaderContent,
  PageHeaderDescription,
  PageHeaderTitle,
} from "@unkey/ui";
import Link from "next/link";
import { UrnText } from "./components/urn-display";
import { CONCEPTS } from "./concepts";
import { perm } from "./lib/mock-data";
import { usePermissionsLab } from "./lib/store";

export default function PermissionsLabOverview() {
  const workspace = useWorkspaceNavigation();
  const lab = usePermissionsLab();
  const scope = { workspaceSlug: workspace.slug };

  return (
    <PageContainer width="full">
      <PageHeader>
        <PageHeaderContent>
          <PageHeaderTitle>Permissions lab</PageHeaderTitle>
          <PageHeaderDescription>
            Ten prototypes for managing URN-based permissions, all running on the same mock ACME
            workspace and a shared client-side grant store. Edits persist locally across pages.
          </PageHeaderDescription>
        </PageHeaderContent>
        <PageHeaderActions>
          <Button variant="outline" onClick={() => lab.reset()}>
            Reset lab data
          </Button>
        </PageHeaderActions>
      </PageHeader>
      <PageBody>
        <div className="flex flex-col gap-6">
          <div className="rounded-lg border border-grayA-4 bg-grayA-2 p-4 flex flex-col gap-2">
            <span className="text-xs uppercase tracking-wide text-gray-10">The grammar</span>
            <UrnText value={perm("keyspaces/ks_payments_prod/keys/*", "read_key")} />
            <p className="text-sm text-gray-11">
              A permission is a resource name plus an action.{" "}
              <span className="font-mono text-xs">*</span> matches exactly one path segment, a
              trailing <span className="font-mono text-xs">**</span> matches the base path and all
              descendants, and grants are allow-only: effective access is always the union of what
              you can see.
            </p>
          </div>
          <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
            {CONCEPTS.map((concept, i) => (
              <Link
                key={concept.segment}
                href={routes.permissionsLab.concept(scope, concept.segment)}
                className="group rounded-lg border border-grayA-4 p-4 hover:border-grayA-7 hover:bg-grayA-2 transition-colors flex flex-col gap-1.5"
              >
                <div className="flex items-center gap-2">
                  <span className="font-mono text-xs text-gray-9">
                    {String(i + 1).padStart(2, "0")}
                  </span>
                  <span className="font-medium text-gray-12 group-hover:text-accent-12">
                    {concept.title}
                  </span>
                </div>
                <p className="text-sm text-gray-11">{concept.pitch}</p>
              </Link>
            ))}
          </div>
        </div>
      </PageBody>
    </PageContainer>
  );
}
