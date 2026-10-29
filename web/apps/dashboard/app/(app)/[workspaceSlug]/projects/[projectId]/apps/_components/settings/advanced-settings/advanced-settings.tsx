import {
  SettingsGroup,
  SettingsGroupContent,
  SettingsGroupTitle,
  SettingsGroups,
  SettingsPageSkeleton,
  Skeleton,
} from "@unkey/ui";
import { OpenapiSpecPath } from "./openapi-spec-path";
import { UpstreamProtocol } from "./upstream-protocol";

export function AdvancedSettings() {
  return (
    <>
      <SettingsGroup>
        <SettingsGroupTitle>API</SettingsGroupTitle>
        <SettingsGroupContent>
          <OpenapiSpecPath />
        </SettingsGroupContent>
      </SettingsGroup>
      <SettingsGroup>
        <SettingsGroupTitle>Networking</SettingsGroupTitle>
        <SettingsGroupContent>
          <UpstreamProtocol />
        </SettingsGroupContent>
      </SettingsGroup>
    </>
  );
}

export function AdvancedSettingsSkeleton() {
  return (
    <SettingsGroups>
      <SettingsPageSkeleton
        groups={[
          {
            title: "API",
            rows: [
              {
                descriptionLines: 0,
                control: <Skeleton className="h-9 w-[436px] max-w-full rounded-md" />,
              },
            ],
          },
          {
            title: "Networking",
            rows: [
              {
                control: (
                  <div className="grid w-full max-w-(--setting-w) grid-cols-2 gap-3">
                    <Skeleton className="h-[94px] rounded-lg" />
                    <Skeleton className="h-[94px] rounded-lg" />
                  </div>
                ),
              },
            ],
          },
        ]}
      />
    </SettingsGroups>
  );
}
