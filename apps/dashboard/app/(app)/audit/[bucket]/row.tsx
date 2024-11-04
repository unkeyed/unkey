"use client";

import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar";
import { Badge } from "@/components/ui/badge";
import { Code } from "@/components/ui/code";
import { TableCell, TableRow } from "@/components/ui/table";
import { cn } from "@/lib/utils";
import { ChevronDown, KeySquare, Minus } from "lucide-react";
import { useState } from "react";

type Props = {
  auditLog: {
    time: number;
    event: string;
    actor: {
      id: string;
      type: string;
      name: string | null;
    };
    location: string | null;
    description: string;
    targets: {
      type: string;
      id: string;
    }[];
  };
  user?: {
    imageUrl: string;
    username: string | null;
    firstName: string | null;
    lastName: string | null;
  };
};
export const Row: React.FC<Props> = ({ auditLog, user }) => {
  const [expandResources, setExpandResources] = useState(false);
  return (
    <>
      <TableRow>
        <TableCell>
          <div className="flex items-center">
            {user ? (
              <div className="flex items-center w-full gap-2 max-sm:m-0 max-sm:gap-1 max-sm:text-xs md:flex-grow">
                <Avatar className="w-6 h-6">
                  <AvatarImage src={user.imageUrl} />
                  <AvatarFallback>{user.username?.slice(0, 2)}</AvatarFallback>
                </Avatar>
                <span className="text-sm text-content whitespace-nowrap">{`${
                  user.firstName ?? ""
                } ${user.lastName ?? ""}`}</span>
              </div>
            ) : (
              <div className="flex items-center w-full gap-2 max-sm:m-0 max-sm:gap-1 max-sm:text-xs md:flex-grow">
                <KeySquare className="w-4 h-4" />
                <span className="font-mono text-xs text-content">{auditLog.actor.id}</span>
              </div>
            )}
          </div>
        </TableCell>
        <TableCell className="flex flex-col gap-1">
          <p className="text-xs text-content-subtle">{auditLog.description}</p>
          <button
            type="button"
            onClick={() => {
              setExpandResources(!expandResources);
            }}
            className="flex items-center gap-2"
          >
            <Badge variant="secondary" className="flex items-center flex-shrink gap-2">
              {auditLog.event}
              <ChevronDown
                className={cn("w-4 h-4 duration-200 transition-all", {
                  "rotate-180": expandResources,
                })}
              />
            </Badge>
          </button>
        </TableCell>
        <TableCell>
          {auditLog.location ? (
            <pre className="text-xs text-content-subtle">{auditLog.location}</pre>
          ) : (
            <Minus className="w-4 h-4 text-content-subtle" />
          )}
        </TableCell>
        <TableCell>
          <div className="flex items-center gap-2">
            <span className="text-sm text-content" suppressHydrationWarning>
              {new Date(auditLog.time).toLocaleDateString()}
            </span>
            <span className="text-xs text-content-subtle" suppressHydrationWarning>
              {new Date(auditLog.time).toLocaleDateString()}
            </span>
          </div>
        </TableCell>
      </TableRow>
      {expandResources ? (
        <TableRow>
          <TableCell />
          <TableCell colSpan={4}>
            <Code className="text-xxs">
              {JSON.stringify(
                auditLog.targets.reduce((acc, t) => {
                  acc[t.type] = t.id;
                  return acc;
                }, {} as any),
                null,
                2,
              )}
            </Code>
          </TableCell>
        </TableRow>
      ) : null}
    </>
  );
};
