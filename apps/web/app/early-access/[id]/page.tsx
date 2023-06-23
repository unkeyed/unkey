import { redirect } from "next/navigation";

/**
 * Legacy, remove after june 2023
 */
export default async function () {
  return redirect("/app");
}
