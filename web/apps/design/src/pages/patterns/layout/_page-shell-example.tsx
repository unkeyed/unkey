import {
  Button,
  PageHeader,
  PageHeaderActions,
  PageHeaderContent,
  PageHeaderDescription,
  PageHeaderTitle,
  PageShell,
  SecondaryNav,
  SecondaryNavGroup,
  SecondaryNavItem,
  SecondaryNavTitle,
} from "@unkey/ui";

type PageShellExampleProps = {
  /** Add a `SecondaryNav` rail to the left of the page. */
  rail?: boolean;
  /** Drop `PageShell` and run the content edge to edge. */
  fullWidth?: boolean;
};

/** A bare page: `PageShell` and `PageHeader` with a placeholder for the body. */
export function PageShellExample({ rail = false, fullWidth = false }: PageShellExampleProps) {
  if (fullWidth) {
    return (
      <div className="pb-8">
        <div className="border-grayA-4 border-b px-4 pt-4 pb-4 lg:px-6 xl:px-10">
          <PageHeader>
            <PageHeaderContent>
              <PageHeaderTitle>Logs</PageHeaderTitle>
              <PageHeaderDescription>Requests across your workspace.</PageHeaderDescription>
            </PageHeaderContent>
            <PageHeaderActions>
              <Button variant="outline">Refresh</Button>
            </PageHeaderActions>
          </PageHeader>
        </div>
        <div className="mt-4 h-48 bg-grayA-3" />
      </div>
    );
  }

  const page = (
    <PageShell className="min-w-0 flex-1 pt-6 pb-8">
      <PageHeader>
        <PageHeaderContent>
          <PageHeaderTitle>General</PageHeaderTitle>
          <PageHeaderDescription>Manage your workspace settings.</PageHeaderDescription>
        </PageHeaderContent>
        <PageHeaderActions>
          <Button variant="outline">Copy ID</Button>
        </PageHeaderActions>
      </PageHeader>
      <div className="mt-4 h-48 rounded-lg bg-grayA-3" />
    </PageShell>
  );

  if (!rail) {
    return page;
  }

  return (
    <div className="flex">
      <SecondaryNav aria-label="Settings" className="md:w-56">
        <SecondaryNavTitle>Settings</SecondaryNavTitle>
        <SecondaryNavGroup>
          <SecondaryNavItem asChild active>
            <a href="#general">General</a>
          </SecondaryNavItem>
          <SecondaryNavItem asChild>
            <a href="#team">Team</a>
          </SecondaryNavItem>
          <SecondaryNavItem asChild>
            <a href="#root-keys">Root Keys</a>
          </SecondaryNavItem>
          <SecondaryNavItem asChild>
            <a href="#billing">Billing</a>
          </SecondaryNavItem>
        </SecondaryNavGroup>
      </SecondaryNav>
      {page}
    </div>
  );
}
