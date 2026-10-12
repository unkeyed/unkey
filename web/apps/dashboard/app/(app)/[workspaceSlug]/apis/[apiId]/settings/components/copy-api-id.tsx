import {
  CopyInput,
  SettingsRow,
  SettingsRowContent,
  SettingsRowDescription,
  SettingsRowHeader,
  SettingsRowTitle,
} from "@unkey/ui";

export const CopyApiId = ({ apiId }: { apiId: string }) => {
  return (
    <SettingsRow>
      <SettingsRowHeader>
        <SettingsRowTitle>API ID</SettingsRowTitle>
        <SettingsRowDescription>
          An identifier for the API, used in some API calls.
        </SettingsRowDescription>
      </SettingsRowHeader>
      <SettingsRowContent>
        <CopyInput value={apiId} aria-label="API ID" className="max-w-(--setting-w)" />
      </SettingsRowContent>
    </SettingsRow>
  );
};
