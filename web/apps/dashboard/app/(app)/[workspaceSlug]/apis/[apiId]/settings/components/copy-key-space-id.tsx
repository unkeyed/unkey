import {
  CopyButton,
  SettingsRow,
  SettingsRowContent,
  SettingsRowDescription,
  SettingsRowHeader,
  SettingsRowTitle,
} from "@unkey/ui";

export const CopyKeySpaceId = ({ keySpaceId }: { keySpaceId: string }) => {
  return (
    <SettingsRow>
      <SettingsRowHeader>
        <SettingsRowTitle>KeySpace ID</SettingsRowTitle>
        <SettingsRowDescription>Identifier for the underlying keyspace.</SettingsRowDescription>
      </SettingsRowHeader>
      <SettingsRowContent>
        <div className="flex max-w-(--setting-w) items-center justify-between rounded-lg border bg-raised px-2 py-2">
          <div className="text-sm text-gray-11">{keySpaceId}</div>
          <CopyButton value={keySpaceId} variant="ghost" toastMessage={keySpaceId} />
        </div>
      </SettingsRowContent>
    </SettingsRow>
  );
};
