"use client";

import { Button } from "@/components/ui/button";
import type { auditLogsDataSchema } from "@/lib/tinybird";
import { download, generateCsv, mkConfig } from "export-to-csv";
import type { z } from "zod";

export function ExportCsv({ data }: { data: z.infer<typeof auditLogsDataSchema>[] }) {
  function csvDownload(rows: z.infer<typeof auditLogsDataSchema>[]) {
    const csvConfig = mkConfig({
      fieldSeparator: ",",
      filename: "unkey-audit-logs", // export file name (without .csv)
      decimalSeparator: ".",
      useKeysAsHeaders: true,
    });
    const formatted = rows
      .map((row) => ({
        ...row,
        actorId: row.actor.id,
        ip: row.context.location,
        userAgent: row.context.userAgent,
        resources: JSON.stringify(row.resources),
      }))
      .map(({ actor, context, resources, ...flattenedRow }) => flattenedRow);
    const csv = generateCsv(csvConfig)(formatted);
    download(csvConfig)(csv);
  }
  return (
    <Button variant="outline" className="text-xs" onClick={() => csvDownload(data)}>
      Export CSV
    </Button>
  );
}
