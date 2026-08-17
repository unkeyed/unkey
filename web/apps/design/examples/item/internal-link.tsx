import { ChartUsage, ChevronRight } from "@unkey/icons";
import { Item, ItemActions, ItemContent, ItemDescription, ItemMedia, ItemTitle } from "@unkey/ui";

export default function InternalLinkItem() {
  return (
    <Item
      variant="outline"
      render={
        <a href="/primitives/item">
          <ItemMedia>
            <ChartUsage />
          </ItemMedia>
          <ItemContent>
            <ItemTitle>Usage</ItemTitle>
            <ItemDescription>Track your spend and usage across Unkey</ItemDescription>
          </ItemContent>
          <ItemActions>
            <ChevronRight />
          </ItemActions>
        </a>
      }
    />
  );
}
