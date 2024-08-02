import { EmptyPlaceholder } from "@/components/dashboard/empty-placeholder";
import { getTenantId } from "@/lib/auth";
import { db } from "@/lib/db";
import { ShieldBan } from "lucide-react";
import Link from "next/link";
import { redirect } from "next/navigation";
import { UsageBanner } from "./banner";
import { DesktopSidebar } from "./desktop-sidebar";
import { MobileSideBar } from "./mobile-sidebar";

interface LayoutProps {
  children: React.ReactNode;
  breadcrumb: React.ReactNode;
}

export default async function Layout({ children, breadcrumb }: LayoutProps) {
  const tenantId = getTenantId();
  const workspace = await db.query.workspaces.findFirst({
    where: (table, { eq }) => eq(table.tenantId, tenantId),
    with: {
      apis: {},
    },
  });
  if (!workspace) {
    return redirect("/apis");
  }

  return (
    <div className="h-[100dvh] relative flex flex-col overflow-hidden bg-background lg:flex-row">
      <UsageBanner workspace={workspace} />

      <MobileSideBar className="lg:hidden" />
      <div className="flex flex-1 overflow-hidden bg-gray-100 dark:bg-gray-950">
        <DesktopSidebar
          workspace={workspace}
          className="isolate hidden lg:flex min-w-[250px] max-w-[250px] bg-[inherit]"
        />

        <div className="isolate bg-background lg:border-l border-t lg:rounded-tl-[0.625rem] border-border w-full overflow-x-auto flex flex-col items-center lg:mt-2">
          <div className="w-full max-w-[1152px] p-4 lg:p-8">
            {workspace.enabled ? (
              <>
                {/* Hacky way to make the breadcrumbs line up with the Teamswitcher on the left, because that also has h12 */}
                {breadcrumb && <div className="block empty:hidden">{breadcrumb}</div>}
                {children}
              </>
            ) : (
              <div className="flex items-center justify-center w-full h-full">
                <EmptyPlaceholder className="border-0">
                  <EmptyPlaceholder.Icon>
                    <ShieldBan />
                  </EmptyPlaceholder.Icon>
                  <EmptyPlaceholder.Title>This workspace is disabled</EmptyPlaceholder.Title>
                  <EmptyPlaceholder.Description>
                    Contact{" "}
                    <Link
                      href={`mailto:support@unkey.dev?body=workspaceId: ${workspace.id}`}
                      className="underline"
                    >
                      support@unkey.dev
                    </Link>
                  </EmptyPlaceholder.Description>
                </EmptyPlaceholder>
              </div>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}
