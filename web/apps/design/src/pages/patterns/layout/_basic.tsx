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

export function BasicExample() {
  return (
    <PageContainer>
      <PageHeader>
        <PageHeaderContent>
          <PageHeaderTitle>General</PageHeaderTitle>
          <PageHeaderDescription>Manage your workspace settings.</PageHeaderDescription>
        </PageHeaderContent>
        <PageHeaderActions>
          <Button variant="outline">Copy ID</Button>
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
