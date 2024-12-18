import { getTenantId } from "@/lib/auth";
import { db } from "@/lib/db";
import { Empty } from "@unkey/ui";
import Link from "next/link";
import { redirect } from "next/navigation";
import { UsageBanner } from "./banner";
import { DesktopSidebar } from "./desktop-sidebar";
import { MobileSideBar } from "./mobile-sidebar";

interface LayoutProps {
  children: React.ReactNode;
}

export default async function Layout({ children }: LayoutProps) {
  const tenantId = await getTenantId();
  const workspace = await db.query.workspaces.findFirst({
    where: (table, { and, eq, isNull }) =>
      and(eq(table.tenantId, tenantId), isNull(table.deletedAtM)),
    with: {
      apis: {
        where: (table, { isNull }) => isNull(table.deletedAtM),
      },
    },
  });
  if (!workspace) {
    return redirect("/apis");
  }

  return (
    <div className="h-[100dvh] relative flex flex-col overflow-hidden bg-background lg:flex-row">
      <UsageBanner workspace={workspace} />

      <MobileSideBar className="lg:hidden" workspace={workspace} />
      <div className="flex flex-1 overflow-hidden bg-gray-100 dark:bg-gray-950">
        <DesktopSidebar
          workspace={workspace}
          className="isolate hidden lg:flex min-w-[250px] max-w-[250px] bg-[inherit]"
        />

        <div
          className="isolate bg-background lg:border-l border-t lg:rounded-tl-[0.625rem] border-border w-full overflow-x-auto flex flex-col items-center lg:mt-2"
          id="layout-wrapper"
        >
          <div className="w-full">
            {workspace.enabled ? (
              children
            ) : (
              <div className="flex items-center justify-center w-full h-full">
                <Empty>
                  <Empty.Icon />
                  <Empty.Title>This workspace is disabled</Empty.Title>
                  <Empty.Description>
                    Contact{" "}
                    <Link
                      href={`mailto:support@unkey.dev?body=workspaceId: ${workspace.id}`}
                      className="underline"
                    >
                      support@unkey.dev
                    </Link>
                  </Empty.Description>
                </Empty>
              </div>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}
