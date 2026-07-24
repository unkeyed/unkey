"use client";
import { CopyableIDButton } from "@/components/navigation/copyable-id-button";
import {
  PageContainer,
  PageHeader,
  PageHeaderActions,
  PageHeaderContent,
  PageHeaderTitle,
} from "@unkey/ui";
import { use } from "react";
import { OverrideIdentifierAction } from "../override-identifier-action";
import { OverridesTable } from "./overrides-table";

export default function OverridePage(props: {
  params: Promise<{ namespaceId: string }>;
}) {
  const params = use(props.params);
  const { namespaceId } = params;

  return (
    <PageContainer width="full">
      <PageHeader>
        <PageHeaderContent>
          <PageHeaderTitle>Overrides</PageHeaderTitle>
        </PageHeaderContent>
        <PageHeaderActions>
          <OverrideIdentifierAction namespaceId={namespaceId} />
          <CopyableIDButton value={namespaceId} />
        </PageHeaderActions>
      </PageHeader>
      <OverridesTable namespaceId={namespaceId} />
    </PageContainer>
  );
}
