"use client";

import type { Interval } from "@/app/(app)/ratelimits/[namespaceId]/filters";
import { Button } from "@/components/ui/button";
import { ScrollArea } from "@/components/ui/scroll-area";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import type { getLatestVerifications } from "@/lib/tinybird";
import { cn } from "@/lib/utils";
import { format } from "date-fns";
import { Eye, EyeOff } from "lucide-react";
import { useState } from "react";

const CELL_CLASS = "py-[2px] text-xs leading-[0.5rem]";
const YELLOW_STATES = ["RATE_LIMITED", "EXPIRED", "USAGE_EXCEEDED"];
const RED_STATES = ["DISABLED", "FORBIDDEN", "INSUFFICIENT_PERMISSIONS"];

type LatestVerifications = Awaited<ReturnType<typeof getLatestVerifications>>["data"];

type Props = {
  verifications: LatestVerifications;
  interval: Interval;
};

export const VerificationTable = ({ verifications, interval }: Props) => {
  const [showIp, setShowIp] = useState(false);

  if (verifications.length === 0) {
    return (
      <div className="relative">
        <div className="w-full flex items-center justify-center bg-background">
          <div className="text-center">
            <h3 className="mt-6 text-xl font-semibold">Not used</h3>
            <p className="text-content-subtle mb-8 mt-2 text-center text-sm font-normal leading-6">
              This key was not used in the last {interval}
            </p>
          </div>
        </div>
      </div>
    );
  }

  return (
    <ScrollArea className="h-[600px]">
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead className="font-mono text-xs">Time</TableHead>
            <TableHead className="font-mono text-xs">User Agent</TableHead>
            <TableHead className="flex h-full items-center text-xs font-mono">
              IP Address{" "}
              <Button
                onClick={() => {
                  setShowIp(!showIp);
                }}
                size="icon"
                variant="link"
              >
                {showIp ? <Eye className="h-4 w-4" /> : <EyeOff className="h-4 w-4" />}
              </Button>
            </TableHead>
            <TableHead className="font-mono text-xs">Region</TableHead>
            <TableHead className="font-mono text-xs p-0">Result</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody className={"font-mono"}>
          {verifications.map((verification, i) => {
            /**
             * Instead of rounding every row individually, we want to round consecutive colored rows together.
             * For example:
             * ╭──────╮
             * │ row1     │
             * ╰──────╯
             * ╭──────╮
             * │ row2     │
             * ╰──────╯
             *
             * Becomes this
             *
             * ╭──────╮
             * │ row1     │
             * │ row2     │
             * ╰──────╯
             */
            const isStartOfColoredBlock =
              verification.outcome !== "VALID" &&
              (i === 0 || verifications[i - 1].outcome === "VALID");
            const isEndOfColoredBlock =
              verification.outcome !== "VALID" &&
              (i === verifications.length - 1 || verifications[i + 1].outcome === "VALID");

            return (
              <TableRow
                key={`${i}-${verification.ipAddress}`}
                className={cn({
                  "bg-amber-2 text-amber-11  hover:bg-amber-3": YELLOW_STATES.includes(
                    verification.outcome,
                  ),
                  "bg-red-2 text-red-11  hover:bg-red-3": RED_STATES.includes(verification.outcome),
                })}
              >
                <TableCell
                  className={cn(CELL_CLASS, "whitespace-nowrap", {
                    "rounded-tl-md": isStartOfColoredBlock,
                    "rounded-bl-md": isEndOfColoredBlock,
                  })}
                >
                  {format(verification.time, "MMM dd HH:mm:ss.SS")}
                </TableCell>
                <TableCell className={cn(CELL_CLASS, "max-w-[150px] truncate")}>
                  {verification.userAgent}
                </TableCell>
                <TableCell className={cn(CELL_CLASS, "font-mono ph-no-capture")}>
                  {showIp
                    ? verification.ipAddress
                    : verification.ipAddress.replace(/[a-z0-9]/g, "*")}
                </TableCell>
                <TableCell className={CELL_CLASS}>{verification.region}</TableCell>
                <TableCell
                  className={cn(CELL_CLASS, "p-2 pl-0", {
                    "rounded-tr-md": isStartOfColoredBlock,
                    "rounded-br-md": isEndOfColoredBlock,
                  })}
                >
                  {verification.outcome}
                </TableCell>
              </TableRow>
            );
          })}
        </TableBody>
      </Table>
    </ScrollArea>
  );
};
