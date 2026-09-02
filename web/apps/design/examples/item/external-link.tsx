import {
  Item,
  ItemActions,
  ItemContent,
  ItemDescription,
  ItemMedia,
  ItemTitle,
  VisuallyHidden,
} from "@unkey/ui";
import { IconArrowUpRightOutline18, IconBookOpenOutline18 } from "nucleo-ui-outline-18";

export default function ExternalLinkItem() {
  return (
    <Item
      variant="outline"
      render={
        <a href="https://unkey.com/docs" target="_blank" rel="noopener noreferrer">
          <ItemMedia>
            <IconBookOpenOutline18 />
          </ItemMedia>
          <ItemContent>
            <ItemTitle>
              Documentation
              <VisuallyHidden> (opens in a new tab)</VisuallyHidden>
            </ItemTitle>
            <ItemDescription>How plans, usage and invoices work</ItemDescription>
          </ItemContent>
          <ItemActions>
            <IconArrowUpRightOutline18 />
          </ItemActions>
        </a>
      }
    />
  );
}
