"use client";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { Check, Eye, EyeOff } from "lucide-react";
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
};

export const AccessTable: React.FC<Props> = ({ verifications }) => {
  const [showIp, setShowIp] = useState(false);
  return (
    <Table>
      <TableHeader>
        <TableRow>
          <TableHead>Time</TableHead>
          <TableHead>Resource</TableHead>
          <TableHead>User Agent</TableHead>
          <TableHead className="flex h-full items-center">
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
          <TableHead>Region</TableHead>
          <TableHead>Valid</TableHead>
        </TableRow>
      </TableHeader>
      <TableBody>
        {verifications.map((verification, i) => (
          // biome-ignore lint/suspicious/noArrayIndexKey: I got nothing better right now
          <TableRow key={i}>
            <TableCell className="flex flex-col">
              <span className="text-content">{new Date(verification.time).toDateString()}</span>
              <span className="text-content-subtle text-xs">
                {new Date(verification.time).toTimeString().split("(").at(0)}
              </span>
            </TableCell>
            <TableCell>{verification.requestedResource}</TableCell>
            <TableCell className="max-w-sm truncate">{verification.userAgent}</TableCell>
            <TableCell className="ph-no-capture font-mono">
              {showIp ? verification.ipAddress : verification.ipAddress.replace(/[a-z0-9]/g, "*")}
            </TableCell>
            <TableCell>{verification.region}</TableCell>
            <TableCell>
              {verification.usageExceeded ? (
                <Badge>Usage Exceede</Badge>
              ) : verification.ratelimited ? (
                <Badge>Ratelimited</Badge>
              ) : (
                <Check />
              )}
            </TableCell>
          </TableRow>
        ))}
      </TableBody>
    </Table>
  );
};
