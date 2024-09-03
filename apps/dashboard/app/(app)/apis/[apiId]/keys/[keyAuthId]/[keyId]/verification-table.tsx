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
import { cn } from "@/lib/utils";
import { format } from "date-fns";
import { Eye, EyeOff } from "lucide-react";
import { JetBrains_Mono } from "next/font/google";
import { useState } from "react";

type Props = {
  verifications: {
    time: number;
    requestedResource?: string;
    ipAddress: string;
    region: string;
    userAgent: string;
    usageExceeded: boolean;
    ratelimited: boolean;
  }[];
  interval: Interval;
};

const jetbrains_mono = JetBrains_Mono({
  subsets: ["latin"],
  display: "swap",
  weight: "300",
});

const CELL_CLASS = "py-[2px] text-xs leading-[0.5rem]";

export const VerificationTable = ({ verifications, interval }: Props) => {
  const [showIp, setShowIp] = useState(false);

  if (verifications.length === 0) {
    return (
      <div className="relative">
        {verifications.length === 0 && (
          <div className="w-full flex items-center justify-center bg-background">
            <div className="text-center">
              <h3 className="mt-6 text-xl font-semibold">Not used</h3>
              <p className="text-content-subtle mb-8 mt-2 text-center text-sm font-normal leading-6">
                This key was not used in the last {interval}
              </p>
            </div>
          </div>
        )}
      </div>
    );
  }

  return (
    <ScrollArea className="h-[600px]">
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead
              variant="bottomBorder"
              className={cn(jetbrains_mono.className, "text-[13px]")}
            >
              Time
            </TableHead>
            <TableHead
              variant="bottomBorder"
              className={cn(jetbrains_mono.className, "text-[13px]")}
            >
              Resource
            </TableHead>
            <TableHead
              variant="bottomBorder"
              className={cn(jetbrains_mono.className, "text-[13px]")}
            >
              User Agent
            </TableHead>
            <TableHead
              variant="bottomBorder"
              className={cn("flex h-full items-center text-[13px]", jetbrains_mono.className)}
            >
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
            <TableHead
              variant="bottomBorder"
              className={cn(jetbrains_mono.className, "text-[13px]")}
            >
              Region
            </TableHead>
            <TableHead
              variant="bottomBorder"
              className={cn(jetbrains_mono.className, "text-[13px]")}
            >
              Result
            </TableHead>
          </TableRow>
        </TableHeader>
        <TableBody className={jetbrains_mono.className}>
          {verifications.map((verification, i) => (
            <TableRow
              key={`${i}-${verification.ipAddress}`}
              className={cn(
                verification.ratelimited
                  ? "bg-amber-2 text-amber-11 rounded-[5px] hover:bg-amber-3"
                  : "",
                verification.usageExceeded
                  ? "bg-red-2 text-red-11 rounded-[5px] hover:bg-red-3"
                  : "",
              )}
            >
              <TableCell className={cn(CELL_CLASS, "whitespace-nowrap rounded-l-[5px]")}>
                {format(verification.time, "MMM dd HH:mm:ss.SS")}
              </TableCell>
              <TableCell className={cn(CELL_CLASS, "max-w-[200px] truncate")}>
                {verification.requestedResource}
              </TableCell>
              <TableCell className={cn(CELL_CLASS, "max-w-[150px] truncate")}>
                {verification.userAgent}
              </TableCell>
              <TableCell className={cn(CELL_CLASS, "font-mono ph-no-capture")}>
                {showIp ? verification.ipAddress : verification.ipAddress.replace(/[a-z0-9]/g, "*")}
              </TableCell>
              <TableCell className={CELL_CLASS}>{verification.region}</TableCell>
              <TableCell className={cn(CELL_CLASS, "p-2 rounded-r-[5px]")}>
                {verification.usageExceeded
                  ? "Usage Exceeded"
                  : verification.ratelimited
                    ? "Ratelimited"
                    : "Verified"}
              </TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </ScrollArea>
  );
};
