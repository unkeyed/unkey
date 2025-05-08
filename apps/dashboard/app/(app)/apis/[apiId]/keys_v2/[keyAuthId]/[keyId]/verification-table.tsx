"use client";

import { ScrollArea } from "@/components/ui/scroll-area";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import type { clickhouse } from "@/lib/clickhouse";
import { cn } from "@/lib/utils";
import { format } from "date-fns";

const CELL_CLASS = "py-[2px] text-xs leading-[0.5rem]";
const YELLOW_STATES = ["RATE_LIMITED", "EXPIRED", "USAGE_EXCEEDED"];
const RED_STATES = ["DISABLED", "FORBIDDEN", "INSUFFICIENT_PERMISSIONS"];

type LatestVerifications = Awaited<ReturnType<typeof clickhouse.verifications.latest>>;

type Props = {
  verifications: LatestVerifications;
};

export const VerificationTable = ({ verifications }: Props) => {
  return (
    <ScrollArea className="h-[600px]">
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead className="font-mono text-xs">Time</TableHead>
            <TableHead className="font-mono text-xs p-0">Result</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody className={"font-mono"}>
          {verifications.val?.map((verification, i) => {
            /**
             * Instead of rounding every row individually, we want to round consecutive colored rows together.
             * For example:
             * ╭──────╮
             * │ row1 │
             * ╰──────╯
             * ╭──────╮
             * │ row2 │
             * ╰──────╯
             *
             * Becomes this
             *
             * ╭──────╮
             * │ row1 │
             * │ row2 │
             * ╰──────╯
             */
            const isStartOfColoredBlock =
              verification.outcome !== "VALID" &&
              (i === 0 || verifications.val[i - 1].outcome === "VALID");
            const isEndOfColoredBlock =
              verification.outcome !== "VALID" &&
              (i === verifications.val.length - 1 || verifications.val[i + 1].outcome === "VALID");

            return (
              <TableRow
                key={`${verification.time}-${i}`}
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
