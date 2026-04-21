import { Gauge, Gear, Key } from "@unkey/icons";
import {
  Button,
  Input,
  SettingCard,
  SettingCardGroup,
  SettingsDangerZone,
  SettingsZoneRow,
} from "@unkey/ui";
import { Preview } from "../../../components/Preview";

export function AnatomyExample() {
  return (
    <Preview>
      <div className="w-full max-w-2xl">
        <SettingCard
          title="Workspace name"
          description="The display name ACME sees across the dashboard."
        >
          <Input defaultValue="ACME Production" className="w-full" />
        </SettingCard>
      </div>
    </Preview>
  );
}

export function WithIconExample() {
  return (
    <Preview>
      <div className="w-full max-w-2xl">
        <SettingCard
          title="API rate limit"
          description="Maximum verifications per second before requests are dropped."
          icon={<Gauge className="text-gray-11" />}
        >
          <Input
            type="number"
            defaultValue={1000}
            className="w-full text-right"
          />
        </SettingCard>
      </div>
    </Preview>
  );
}

export function GroupExample() {
  return (
    <Preview>
      <div className="w-full max-w-2xl">
        <SettingCardGroup>
          <SettingCard
            title="Workspace name"
            description="The display name ACME sees across the dashboard."
            icon={<Gear className="text-gray-11" />}
          >
            <Input defaultValue="ACME Production" className="w-full" />
          </SettingCard>
          <SettingCard
            title="API rate limit"
            description="Verifications per second before requests are dropped."
            icon={<Gauge className="text-gray-11" />}
          >
            <Input
              type="number"
              defaultValue={1000}
              className="w-full text-right"
            />
          </SettingCard>
          <SettingCard
            title="Root key"
            description="Used by CI to provision keys. Rotate it if it leaks."
            icon={<Key className="text-gray-11" />}
          >
            <Button variant="outline" className="ml-auto">
              Rotate
            </Button>
          </SettingCard>
        </SettingCardGroup>
      </div>
    </Preview>
  );
}

export function ExpandableExample() {
  return (
    <Preview>
      <div className="w-full max-w-2xl">
        <SettingCard
          title="Email notifications"
          description="Configure which workspace events send email to admins."
          icon={<Gear className="text-gray-11" />}
          expandable={
            <div className="px-4 py-5 text-sm text-gray-11 space-y-3">
              <p>Send email when:</p>
              <ul className="list-disc pl-5 space-y-1">
                <li>A root key is created or rotated</li>
                <li>Monthly verification quota exceeds 80%</li>
                <li>A deployment fails to roll out</li>
              </ul>
            </div>
          }
        >
          <Button variant="outline" className="ml-auto">
            3 enabled
          </Button>
        </SettingCard>
      </div>
    </Preview>
  );
}

export function DangerZoneExample() {
  return (
    <Preview>
      <div className="w-full max-w-2xl">
        <SettingsDangerZone>
          <SettingsZoneRow
            title="Transfer workspace"
            description="Move ACME to a different billing owner. Existing keys keep working."
            action={{
              label: "Transfer",
              onClick: () => {},
            }}
          />
          <SettingsZoneRow
            title="Delete workspace"
            description="Permanently deletes ACME and every key it owns. Cannot be undone."
            action={{
              label: "Delete",
              onClick: () => {},
            }}
          />
        </SettingsDangerZone>
      </div>
    </Preview>
  );
}
