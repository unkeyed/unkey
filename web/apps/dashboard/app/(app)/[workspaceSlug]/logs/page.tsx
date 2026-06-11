import {
  PageBody,
  PageContainer,
  PageHeader,
  PageHeaderContent,
  PageHeaderTitle,
} from "@unkey/ui";
import { LogsClient } from "./components/logs-client";

export default function Page() {
  return (
    <PageContainer width="full">
      <PageHeader>
        <PageHeaderContent>
          <PageHeaderTitle>Logs</PageHeaderTitle>
        </PageHeaderContent>
      </PageHeader>
      <PageBody>
        <LogsClient />
      </PageBody>
    </PageContainer>
  );
}
