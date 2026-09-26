"use client";

import { PageBody, PageContainer, PageHeader, PageHeaderContent, PageHeaderTitle } from "@unkey/ui";
import { ProtoCanvas } from "../proto-canvas";
import { HelpRow, type VariantProps } from "../shared";

export function LanesVariant(p: VariantProps) {
  return (
    <PageContainer>
      <PageHeader>
        <PageHeaderContent>
          <PageHeaderTitle>{p.title}</PageHeaderTitle>
        </PageHeaderContent>
      </PageHeader>
      <PageBody className="flex flex-col gap-6">
        <ProtoCanvas {...p} options={{ arrangement: "lanes", heightClass: "h-[640px]" }} />
        <HelpRow />
      </PageBody>
    </PageContainer>
  );
}
