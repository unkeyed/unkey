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
}

export default async function Layout({ children }: LayoutProps) {
  const tenantId = getTenantId();
  const workspace = await db.query.workspaces.findFirst({
    where: (table, { and, eq, isNull }) =>
      and(eq(table.tenantId, tenantId), isNull(table.deletedAt)),
    with: {
      apis: {
        where: (table, { isNull }) => isNull(table.deletedAt),
      },
    },
  });
  if (!workspace) {
    return redirect("/apis");
  }

  return (
    <>
      <div className="relative flex flex-col min-h-screen lg:flex-row bg-white dark:bg-black">
        <UsageBanner workspace={workspace} />
        <DesktopSidebar workspace={workspace} className="hidden lg:flex min-w-[250px] max-w-[250px]" />
        <MobileSideBar className="lg:hidden" />
        <div className="bg-gray-100 dark:bg-gray-950 lg:border-l border-t lg:rounded-tl-[0.625rem] border-border w-full overflow-x-auto flex flex-col items-center lg:mt-2">
          <div className="w-full max-w-[1024px] p-4 lg:p-8">
            {workspace.enabled ? (
              children
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
    </>
  );
}
