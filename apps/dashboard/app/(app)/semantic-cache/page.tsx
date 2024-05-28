import { PageHeader } from "@/components/dashboard/page-header";
import { getTenantId } from "@/lib/auth";
import { db } from "@/lib/db";
import { redirect } from "next/navigation";
import Form from "./form";
import Logs from "./logs";

export default async function SemanticCachePage() {
  return (
    <div>
      <Logs />
    </div>
  );
}
