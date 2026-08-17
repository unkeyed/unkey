import { ChartUsage, Gauge, Key2 } from "@unkey/icons";
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

export default function SettingsGroupExample() {
  return (
    <SettingCardGroup>
      <SettingCard
        title="Spend cap"
        description="Compute stops when the month reaches this amount."
        icon={<Key2 />}
        contentWidth="w-fit"
        className="w-full flex-row items-center justify-between"
      >
        <Button variant="outline">Edit</Button>
      </SettingCard>
      <Item>
        <ItemMedia>
          <ChartUsage />
        </ItemMedia>
        <ItemContent>
          <ItemTitle>Key verifications</ItemTitle>
          <ItemDescription>1.2M of 2M included</ItemDescription>
        </ItemContent>
        <ItemActions className="tabular-nums">$0.00</ItemActions>
      </Item>
      <Item>
        <ItemMedia>
          <Gauge />
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
