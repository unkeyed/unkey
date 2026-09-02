import {
  Button,
  Item,
  ItemActions,
  ItemContent,
  ItemDescription,
  ItemGroup,
  ItemHeader,
  ItemMedia,
  ItemSeparator,
  ItemTitle,
} from "@unkey/ui";
import { IconChartUsageOutline18, IconGaugeOutline18 } from "nucleo-ui-outline-18";

export default function ItemGroupExample() {
  return (
    <ItemGroup variant="outline">
      <ItemHeader>
        <ItemContent>
          <ItemTitle>Usage this month</ItemTitle>
          <ItemDescription>Resets on 1 September</ItemDescription>
        </ItemContent>
        <ItemActions>
          <Button variant="outline">Manage plan</Button>
        </ItemActions>
      </ItemHeader>
      <ItemSeparator />
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
    </ItemGroup>
  );
}
