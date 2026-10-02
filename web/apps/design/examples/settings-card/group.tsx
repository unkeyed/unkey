import { IconEarthOutline18, IconKey2Outline18 } from "@unkey/icons";
import { Button, SettingCard, SettingCardGroup } from "@unkey/ui";

export default function SettingCardGroupExample() {
  return (
    <SettingCardGroup>
      <SettingCard
        title="Workspace name"
        description="Shown across the dashboard and in invoices."
        icon={<IconKey2Outline18 />}
        contentWidth="w-fit"
        className="w-full flex-row items-center justify-between"
      >
        <Button variant="outline">Edit</Button>
      </SettingCard>
      <SettingCard
        title="Custom domain"
        description="Serve your gateway from a domain you own."
        icon={<IconEarthOutline18 />}
        contentWidth="w-fit"
        className="w-full flex-row items-center justify-between"
      >
        <Button variant="outline">Add domain</Button>
      </SettingCard>
    </SettingCardGroup>
  );
}
