import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { getRatelimitLastUsed } from "@/lib/tinybird";
import { ChevronRight, Minus } from "lucide-react";
import ms from "ms";
import Link from "next/link";

type Props = {
  workspaceId: string;
  namespaceId: string;
  ratelimits: {
    id: string;
    identifier: string;
    limit: number;
    duration: number;
  }[];
};

export const Table: React.FC<Props> = async ({ workspaceId, namespaceId, ratelimits }) => {
  return (
    <ul className="flex flex-col overflow-hidden border divide-y rounded-lg divide-border bg-background border-border">
      {ratelimits.map((rl) => (
        <Link
          href={`/app/ratelimits/${namespaceId}/overrides/${rl.id}`}
          key={rl.id}
          className="grid items-center grid-cols-12 px-4 py-2 duration-250 hover:bg-background-subtle "
        >
          <div className="flex flex-col items-start col-span-5 ">
            <span className="text-sm text-content">{rl.identifier}</span>
            <pre className="text-xs text-content-subtle">{rl.id}</pre>
          </div>

          <div className="flex items-center col-span-4 gap-2">
            <Badge variant="secondary">{Intl.NumberFormat().format(rl.limit)} requests</Badge>
            <span className="text-content-subtle">/</span>
            <Badge variant="secondary">{ms(rl.duration)}</Badge>
          </div>

          <div className="flex items-center col-span-2 gap-2">
            <LastUsed
              workspaceId={workspaceId}
              namespaceId={namespaceId}
              identifier={rl.identifier}
            />
          </div>

          <div className="flex items-center justify-end col-span-1">
            <Button variant="ghost">
              <ChevronRight className="w-4 h-4" />
            </Button>
          </div>
        </Link>
      ))}
    </ul>
  );
};

const LastUsed: React.FC<{ workspaceId: string; namespaceId: string; identifier: string }> =
  async ({ workspaceId, namespaceId, identifier }) => {
    const lastUsed = await getRatelimitLastUsed({
      workspaceId,
      namespaceId,
      identifier: [identifier],
    });

    const unixMilli = lastUsed.data.at(0)?.lastUsed;
    if (unixMilli) {
      return <span className="text-sm text-content-subtle">{ms(Date.now() - unixMilli)} ago</span>;
    }
    return <Minus className="w-4 h-4 text-content-subtle" />;
  };
