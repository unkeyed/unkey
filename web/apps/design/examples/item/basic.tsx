import {
  Button,
  Item,
  ItemActions,
  ItemContent,
  ItemDescription,
  ItemMedia,
  ItemTitle,
} from "@unkey/ui";
import {
  IconBookOpenOutline18,
  IconChartUsageOutline18,
  IconChevronRightOutline18,
} from "nucleo-ui-outline-18";

export default function BasicItem() {
  return (
    <div className="flex flex-col gap-4">
      <Item>
        <ItemContent>
          <ItemTitle>Basic Item</ItemTitle>
          <ItemDescription>A simple item with title and description.</ItemDescription>
        </ItemContent>
        <ItemActions>
          <Button variant="outline">Action</Button>
        </ItemActions>
      </Item>
      <Item
        variant="outline"
        render={
          <a href="/primitives/item">
            <ItemMedia>
              <IconChartUsageOutline18 />
            </ItemMedia>
            <ItemContent>
              <ItemTitle>Usage</ItemTitle>
              <ItemDescription>Track your spend and usage across Unkey.</ItemDescription>
            </ItemContent>
            <ItemActions>
              <IconChevronRightOutline18 />
            </ItemActions>
          </a>
        }
      />
      <Item
        variant="outline"
        render={
          <a href="/primitives/item">
            <ItemMedia>
              <IconBookOpenOutline18 />
            </ItemMedia>
            <ItemContent>
              <ItemTitle>Docs</ItemTitle>
              <ItemDescription>How plans, usage and invoices work.</ItemDescription>
            </ItemContent>
            <ItemActions>
              <IconChevronRightOutline18 />
            </ItemActions>
          </a>
        }
      />
    </div>
  );
}
