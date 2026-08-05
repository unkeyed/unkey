import { BookOpen, ChartUsage, ChevronRight } from "@unkey/icons";
import {
  Button,
  Item,
  ItemActions,
  ItemContent,
  ItemDescription,
  ItemMedia,
  ItemTitle,
} from "@unkey/ui";

export function ItemLink() {
  return (
    <div className="flex w-full max-w-md flex-col gap-4">
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
              <ChartUsage />
            </ItemMedia>
            <ItemContent>
              <ItemTitle>Usage</ItemTitle>
              <ItemDescription>Track your spend and usage across Unkey.</ItemDescription>
            </ItemContent>
            <ItemActions>
              <ChevronRight />
            </ItemActions>
          </a>
        }
      />
      <Item
        variant="outline"
        render={
          <a href="/primitives/item">
            <ItemMedia>
              <BookOpen />
            </ItemMedia>
            <ItemContent>
              <ItemTitle>Docs</ItemTitle>
              <ItemDescription>How plans, usage and invoices work.</ItemDescription>
            </ItemContent>
            <ItemActions>
              <ChevronRight />
            </ItemActions>
          </a>
        }
      />
    </div>
  );
}
