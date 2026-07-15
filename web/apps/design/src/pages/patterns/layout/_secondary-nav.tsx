import {
  PageBody,
  PageContainer,
  PageHeader,
  PageHeaderContent,
  PageHeaderTitle,
  SecondaryNav,
  SecondaryNavGroup,
  SecondaryNavItem,
  SecondaryNavTitle,
} from "@unkey/ui";

export function SecondaryNavExample() {
  return (
    <div className="flex w-full flex-1 flex-col md:flex-row">
      <SecondaryNav aria-label="Settings" className="md:w-56">
        <SecondaryNavTitle>Settings</SecondaryNavTitle>
        <SecondaryNavGroup>
          <SecondaryNavItem asChild active>
            <a href="#general">General</a>
          </SecondaryNavItem>
          <SecondaryNavItem asChild>
            <a href="#team">Team</a>
          </SecondaryNavItem>
        </SecondaryNavGroup>
      </SecondaryNav>
      <div className="min-w-0 flex-1">
        <PageContainer>
          <PageHeader>
            <PageHeaderContent>
              <PageHeaderTitle>General</PageHeaderTitle>
            </PageHeaderContent>
          </PageHeader>
          <PageBody>
            <div className="mt-4 flex h-48 w-full items-center justify-center rounded-lg bg-grayA-3 font-mono text-gray-11 text-xs">
              Your content here
            </div>
          </PageBody>
        </PageContainer>
      </div>
    </div>
  );
}
