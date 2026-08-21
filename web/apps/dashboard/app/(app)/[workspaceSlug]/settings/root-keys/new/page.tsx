"use client";

import { useFlag } from "@/lib/flags/provider";
import { PageBody, PageContainer, PageHeader, PageHeaderContent, PageHeaderTitle } from "@unkey/ui";
import { notFound } from "next/navigation";
import { BuilderShell } from "./components/builder-shell";

export default function NewRootKeyPage() {
  const rootKeyBuilder = useFlag("rootKeyBuilder");

  if (!rootKeyBuilder) {
    notFound();
  }

  return (
    <PageContainer>
      <PageHeader>
        <PageHeaderContent>
          <PageHeaderTitle>New Root Key</PageHeaderTitle>
        </PageHeaderContent>
      </PageHeader>
      <PageBody>
        <BuilderShell />
      </PageBody>
    </PageContainer>
  );
}
