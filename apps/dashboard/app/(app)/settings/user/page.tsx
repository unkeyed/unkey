import { Navbar as SubMenu } from "@/components/dashboard/navbar";
import { Navigation } from "@/components/navigation/navigation";
import { PageContent } from "@/components/page-content";
import { getTenantId } from "@/lib/auth";
import { db } from "@/lib/db";
import { Gear } from "@unkey/icons";
import { redirect } from "next/navigation";
import { navigation } from "../constants";
import { UpdateTheme } from "./update-theme";
// import { UpdateUserEmail } from "./update-user-email";
// import { UpdateUserImage } from "./update-user-image";
// import { UpdateUserName } from "./update-user-name";

/**
 * TODO: WorkOS doesn't support changing user email, image, and has no usernames
 */

export const revalidate = 0;

export default async function SettingsPage() {
  const tenantId = await getTenantId();

  const workspace = await db.query.workspaces.findFirst({
    where: (table, { and, eq, isNull }) =>
      and(eq(table.tenantId, tenantId), isNull(table.deletedAtM)),
  });
  if (!workspace) {
    return redirect("/new");
  }

  return (
    <div>
      <Navigation href="/settings/user" name="Settings" icon={<Gear />} />
      <PageContent>
        <SubMenu navigation={navigation} segment="user" />

        <div className="mb-20 flex flex-col gap-8 mt-8">
          {/* <UpdateUserName />
          <UpdateUserEmail />
          <UpdateUserImage /> */}
          <UpdateTheme />
        </div>
      </PageContent>
    </div>
  );
}
