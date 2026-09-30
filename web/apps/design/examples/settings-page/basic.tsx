import {
  FormInput,
  PageContainer,
  PageHeader,
  PageHeaderContent,
  PageHeaderTitle,
  SecondaryNav,
  SecondaryNavGroup,
  SecondaryNavItem,
  SecondaryNavTitle,
  SettingsDangerZone,
  SettingsGroup,
  SettingsGroups,
  SettingsRow,
  SettingsZoneRow,
} from "@unkey/ui";

export default function SettingsPageExample() {
  return (
    <div className="flex w-full flex-1 flex-col md:flex-row">
      <SecondaryNav aria-label="App settings">
        <SecondaryNavTitle>App Settings</SecondaryNavTitle>
        <SecondaryNavGroup>
          <SecondaryNavItem href="#build">Build</SecondaryNavItem>
          <SecondaryNavItem href="#runtime" active>
            Runtime
          </SecondaryNavItem>
          <SecondaryNavItem href="#advanced">Advanced</SecondaryNavItem>
        </SecondaryNavGroup>
      </SecondaryNav>
      <div className="min-w-0 flex-1">
        <PageContainer>
          <PageHeader className="max-w-[920px] px-6">
            <PageHeaderContent>
              <PageHeaderTitle>Runtime</PageHeaderTitle>
            </PageHeaderContent>
          </PageHeader>
          <SettingsGroups>
            <SettingsGroup>
              <SettingsRow title="Port" description="Port your application listens on.">
                <FormInput defaultValue="8080" />
              </SettingsRow>
              <SettingsRow title="Command" description="The command that starts your application.">
                <FormInput defaultValue="node server.js" />
              </SettingsRow>
            </SettingsGroup>
            <SettingsDangerZone>
              <SettingsZoneRow
                title="Delete this app"
                description="Once you delete an app, there is no going back. Please be certain."
                action={{ label: "Delete this app", onClick: () => {} }}
              />
            </SettingsDangerZone>
          </SettingsGroups>
        </PageContainer>
      </div>
    </div>
  );
}
