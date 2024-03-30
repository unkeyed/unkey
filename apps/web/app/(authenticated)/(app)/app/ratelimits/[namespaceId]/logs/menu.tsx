"use client";
import { Copy, Filter, MoreHorizontal, UserRoundCog } from "lucide-react";

import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { toast } from "@/components/ui/toaster";
import Link from "next/link";
import { parseAsArrayOf, parseAsString, useQueryState } from "nuqs";

type Props = {
  namespace: { id: string };
  identifier: string;
};

export const Menu: React.FC<Props> = ({ namespace, identifier }) => {
  const [_, setIdentifier] = useQueryState(
    "identifier",
    parseAsArrayOf(parseAsString).withDefault([]).withOptions({
      history: "push",
      shallow: false, // otherwise server components won't notice the change
      clearOnDefault: true,
    }),
  );

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button variant="ghost" size="icon">
          <MoreHorizontal className="w-4 h-4" />
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent className="w-56">
        <DropdownMenuItem
          onClick={() => {
            navigator.clipboard.writeText(identifier);
            toast.success("Copied to clipboard", {
              description: identifier,
            });
          }}
        >
          <Copy className="w-4 h-4 mr-2" />
          <span>Copy identifier</span>
        </DropdownMenuItem>
        <Link href={`/app/ratelimits/${namespace.id}/overrides?identifier=${identifier}`}>
          <DropdownMenuItem>
            <UserRoundCog className="w-4 h-4 mr-2" />
            <span>Override</span>
          </DropdownMenuItem>
        </Link>
        <DropdownMenuItem
          onClick={() => {
            setIdentifier([identifier]);
          }}
        >
          <Filter className="w-4 h-4 mr-2" />
          <span>Filter for identifier</span>
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  );
};
