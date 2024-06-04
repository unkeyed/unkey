import { getTenantId } from "@/lib/auth";
import { db } from "@/lib/db";
import { redirect } from "next/navigation";
import Form from "../form";

export default async function NewSemanticCachePage() {
  return <Form />;
}
