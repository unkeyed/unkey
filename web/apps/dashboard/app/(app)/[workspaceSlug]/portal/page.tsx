import {
  Empty,
  PageBody,
  PageContainer,
  PageHeader,
  PageHeaderContent,
  PageHeaderTitle,
} from "@unkey/ui";

export default function PortalPage() {
  return (
    <PageContainer>
      <PageHeader>
        <PageHeaderContent>
          <PageHeaderTitle>Portal</PageHeaderTitle>
        </PageHeaderContent>
      </PageHeader>
      <PageBody>
        <Empty>
          <Empty.Title>Coming soon</Empty.Title>
          <Empty.Description>
            Portal configuration is on its way. Check back once it ships.
          </Empty.Description>
        </Empty>
      </PageBody>
    </PageContainer>
  );
}
