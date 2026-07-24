import {
  PageBody,
  PageContainer,
  PageHeader,
  PageHeaderActions,
  PageHeaderContent,
  PageHeaderTitle,
} from "@unkey/ui";
import { CreateIdentityDialog } from "./_components/dialogs/create-identity-dialog";
import { IdentitiesClient } from "./_components/identities-client";

export default function Page() {
  return (
    <PageContainer>
      <PageHeader>
        <PageHeaderContent>
          <PageHeaderTitle>Identities</PageHeaderTitle>
        </PageHeaderContent>
        <PageHeaderActions>
          <CreateIdentityDialog />
        </PageHeaderActions>
      </PageHeader>
      <PageBody>
        <IdentitiesClient />
      </PageBody>
    </PageContainer>
  );
}
