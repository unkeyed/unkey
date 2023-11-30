import { redirect } from "next/navigation";
export const runtime = "edge";
export default function OverviewPage() {
  return redirect("/app/apis");
}
