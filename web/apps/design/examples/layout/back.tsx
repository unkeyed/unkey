import {
  Button,
  PageBody,
  PageContainer,
  PageHeader,
  PageHeaderActions,
  PageHeaderBack,
  PageHeaderContent,
  PageHeaderTitle,
} from "@unkey/ui";

export default function LayoutWithBackLink() {
  return (
    <PageContainer>
      <PageHeader>
        <PageHeaderContent>
          <PageHeaderBack href="#">Log Drains</PageHeaderBack>
          <PageHeaderTitle>Acme production logs</PageHeaderTitle>
        </PageHeaderContent>
        <PageHeaderActions>
          <Button variant="outline">Edit</Button>
        </PageHeaderActions>
      </PageHeader>
      <PageBody>
        <div className="mt-4 flex h-44 w-full items-center justify-center rounded-lg bg-grayA-3 font-mono text-gray-11 text-xs">
          Your content here
        </div>
      </PageBody>
    </PageContainer>
  );
}
