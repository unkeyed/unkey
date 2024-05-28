"use client";
import { PageHeader } from "@/components/dashboard/page-header";
import { CreateApiButton } from "./create-api-button";

import { Separator } from "@/components/ui/separator";
import { getTenantId } from "@/lib/auth";
import { and, db, eq, isNull, schema, sql } from "@/lib/db";
import { useApis } from "@/lib/replicache/hooks";
import { Search } from "lucide-react";
import Link from "next/link";
import { redirect } from "next/navigation";
import { ApiList } from "./client";

export const dynamic = "force-dynamic";
export const runtime = "edge";

export default function ApisOverviewPage() {
  const apis = useApis();

  return (
    <div className="">
      <ApiList
        apis={Object.values(apis).map((api) => ({
          ...api,
          keys: [],
        }))}
      />
    </div>
  );
}
