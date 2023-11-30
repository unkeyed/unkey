import { redirect } from "next/navigation";
export const runtime = "edge";
export default function SettingsPage() {
  return redirect("/app/settings/general");
}
