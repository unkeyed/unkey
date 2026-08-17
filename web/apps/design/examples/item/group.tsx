import { ChartUsage, Gauge } from "@unkey/icons";
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
    </ItemGroup>
  );
}
