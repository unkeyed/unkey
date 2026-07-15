import {
  Empty,
  PageBody,
  PageContainer,
  PageHeader,
  PageHeaderContent,
  PageHeaderTitle,
} from "@unkey/ui";

export default function ProjectOverviewPage() {
  return (
    <PageContainer>
      <PageHeader>
        <PageHeaderContent>
          <PageHeaderTitle>Overview</PageHeaderTitle>
        </PageHeaderContent>
      </PageHeader>
      <PageBody>
        <Empty>
          <Empty.Icon />
          <Empty.Title>Project overview</Empty.Title>
          <Empty.Description>An at-a-glance view of this project is coming soon.</Empty.Description>
        </Empty>
      </PageBody>
    </PageContainer>
  );
}
