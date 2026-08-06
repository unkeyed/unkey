import { ArrowUpRight, BookOpen } from "@unkey/icons";
import {
  Item,
  ItemActions,
  ItemContent,
  ItemDescription,
  ItemMedia,
  ItemTitle,
  VisuallyHidden,
} from "@unkey/ui";

export function ExternalLink() {
  return (
    <div className="w-full max-w-md">
      <Item
        variant="outline"
        render={
          <a href="https://unkey.com/docs" target="_blank" rel="noopener noreferrer">
            <ItemMedia>
              <BookOpen />
            </ItemMedia>
            <ItemContent>
              <ItemTitle>
                Documentation
                <VisuallyHidden> (opens in a new tab)</VisuallyHidden>
              </ItemTitle>
              <ItemDescription>How plans, usage and invoices work</ItemDescription>
            </ItemContent>
            <ItemActions>
              <ArrowUpRight />
            </ItemActions>
          </a>
        }
      />
    </div>
  );
}
