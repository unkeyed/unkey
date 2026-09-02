import {
  Button,
  Item,
  ItemActions,
  ItemContent,
  ItemDescription,
  ItemMedia,
  ItemTitle,
  SettingCard,
  SettingCardGroup,
} from "@unkey/ui";
import {
  IconChartUsageOutline18,
  IconGaugeOutline18,
  IconKey2Outline18,
} from "nucleo-ui-outline-18";

export default function SettingsGroupExample() {
  return (
    <SettingCardGroup>
      <SettingCard
        title="Spend cap"
        description="Compute stops when the month reaches this amount."
        icon={<IconKey2Outline18 />}
        contentWidth="w-fit"
        className="w-full flex-row items-center justify-between"
      >
        <Button variant="outline">Edit</Button>
      </SettingCard>
      <Item>
        <ItemMedia>
          <IconChartUsageOutline18 />
        </ItemMedia>
        <ItemContent>
          <ItemTitle>Key verifications</ItemTitle>
          <ItemDescription>1.2M of 2M included</ItemDescription>
        </ItemContent>
        <ItemActions className="tabular-nums">$0.00</ItemActions>
      </Item>
      <Item>
        <ItemMedia>
          <IconGaugeOutline18 />
        </ItemMedia>
        <ItemContent>
          <ItemTitle>Compute</ItemTitle>
          <ItemDescription>18 hours of vCPU time</ItemDescription>
        </ItemContent>
        <ItemActions className="tabular-nums">$4.32</ItemActions>
      </Item>
    </SettingCardGroup>
  );
}
