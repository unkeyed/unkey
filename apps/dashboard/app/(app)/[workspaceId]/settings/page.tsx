import { redirect } from "next/navigation";

export const dynamic = "force-dynamic";

export default function SettingsPage({ params }: { params: { workspaceId: string } }) {
  const { workspaceId } = params;
  if (workspaceId === "new") {
    return redirect("/new");
  }
  if (workspaceId === "settings") {
    return redirect(`/${workspaceId}/settings/general`);
  }
  return redirect(`/${workspaceId}/settings/general`);
}
