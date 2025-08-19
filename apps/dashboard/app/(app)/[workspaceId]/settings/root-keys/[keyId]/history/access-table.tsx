"use client";
import {
  Table,
  TableBody,
  TableCaption,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { Badge } from "@unkey/ui";
import { Check } from "lucide-react";

type Props = {
  verifications: {
    time: number;
    region: string;
    outcome: string;
  }[];
};

export const AccessTable: React.FC<Props> = ({ verifications }) => {
  return (
    <Table>
      {verifications.length === 0 ? (
        <TableCaption className="text-left">This key was not used yet</TableCaption>
      ) : null}
      <TableHeader>
        <TableRow>
          <TableHead>Time</TableHead>
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
              <span className="text-xs text-content-subtle">
                {new Date(verification.time).toTimeString().split("(").at(0)}
              </span>
            </TableCell>
            <TableCell>{verification.region}</TableCell>
            <TableCell>
              {verification.outcome === "VALID" ? <Check /> : <Badge>{verification.outcome}</Badge>}
            </TableCell>
          </TableRow>
        ))}
      </TableBody>
    </Table>
  );
};
