import {
  Button,
  PageBody,
  PageContainer,
  PageHeader,
  PageHeaderActions,
  PageHeaderContent,
  PageHeaderDescription,
  PageHeaderTitle,
} from "@unkey/ui";

export function FullWidthExample() {
  return (
    <PageContainer width="full">
      <PageHeader>
        <PageHeaderContent>
          <PageHeaderTitle>Logs</PageHeaderTitle>
          <PageHeaderDescription>Requests across your workspace.</PageHeaderDescription>
        </PageHeaderContent>
        <PageHeaderActions>
          <Button variant="outline">Refresh</Button>
        </PageHeaderActions>
      </PageHeader>
      <PageBody>
        <div className="flex h-52 w-full items-center justify-center bg-grayA-3 font-mono text-gray-11 text-xs">
          Your content here
        </div>
      </PageBody>
    </PageContainer>
  );
}
