import { Dots } from "@unkey/icons";
import { Button } from "@unkey/ui";
import { cn } from "@unkey/ui/src/lib/utils";
import { TableActionPopover } from "./components/table-action-popover";

export const LogsTableAction = ({
  identifier,
  namespaceId,
}: {
  identifier: string;
  namespaceId: string;
}) => {
  return (
    <TableActionPopover identifier={identifier} namespaceId={namespaceId}>
      <Button
        className={cn(
          "group-data-[state=open]:bg-gray-6 px-2 bg-gray-5 hover:bg-gray-6 group border-none",
        )}
        size="icon"
      >
        <Dots className="group-hover:text-gray-12 text-gray-11" />
      </Button>
    </TableActionPopover>
  );
};
