import { MoreHorizontal, RefreshCw } from "lucide-react";
import { Button } from "~/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "~/components/ui/dropdown-menu";
import { Tooltip, TooltipContent, TooltipTrigger } from "~/components/ui/tooltip";
import { keyStatus } from "./key-status";
import type { Key } from "./schema/keys.schema";

type Props = {
  apiKey: Key;
  onRotate: () => void;
};

export function KeyActions({ apiKey, onRotate }: Props) {
  const expired = keyStatus(apiKey) === "expired";

  return (
    <DropdownMenu>
      <DropdownMenuTrigger
        render={
          <Button
            variant="ghost"
            size="icon"
            aria-label={`Actions for ${apiKey.name ?? apiKey.id}`}
          >
            <MoreHorizontal />
          </Button>
        }
      />
      <DropdownMenuContent align="end" className="min-w-56">
        {expired ? (
          <Tooltip>
            <TooltipTrigger
              render={
                <span>
                  <DropdownMenuItem disabled>
                    <RefreshCw />
                    Rotate key
                  </DropdownMenuItem>
                </span>
              }
            />
            <TooltipContent side="left">Expired keys can't be rotated.</TooltipContent>
          </Tooltip>
        ) : (
          <DropdownMenuItem onClick={onRotate}>
            <RefreshCw />
            Rotate key
          </DropdownMenuItem>
        )}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
