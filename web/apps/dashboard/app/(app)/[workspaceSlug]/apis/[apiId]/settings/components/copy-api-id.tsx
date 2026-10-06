import {
  CopyButton,
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
        <div className="flex max-w-(--setting-w) items-center justify-between rounded-lg border bg-raised px-2 py-2">
          <div className="text-sm text-gray-11">{apiId}</div>
          <CopyButton value={apiId} variant="ghost" toastMessage={apiId} />
        </div>
      </SettingsRowContent>
    </SettingsRow>
  );
};
