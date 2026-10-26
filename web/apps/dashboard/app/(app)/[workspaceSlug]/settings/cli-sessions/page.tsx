import { getAuth } from "@/lib/auth";
import { auth } from "@/lib/auth/server";
import { type CliSessionOwner, CliSessions } from "./cli-sessions";

export default async function CliSessionsSettingsPage() {
  const { orgId, role } = await getAuth();
  let owners: CliSessionOwner[] = [];

  if (role === "admin") {
    try {
      const members = await auth.getOrganizationMemberList(orgId);
      owners = members.data.map(({ user }) => ({
        id: user.id,
        fullName: user.fullName,
        email: user.email,
      }));
    } catch (error) {
      console.error("Failed to load CLI session owners:", error);
    }
  }

  return <CliSessions owners={owners} />;
}
