import { Dots } from "@unkey/icons";
import { Button } from "@unkey/ui";
import { cn } from "@unkey/ui/src/lib/utils";
import { TableActionPopover } from "./components/table-action-popover";

export const LogsTableAction = () => {
  return (
    <TableActionPopover>
      <Button className={cn("group-data-[state=open]:bg-gray-4 px-2")} size="icon">
        <Dots />
      </Button>
    </TableActionPopover>
  );
};
