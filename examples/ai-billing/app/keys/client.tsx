"use client";
import {
  Card,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeaderCell,
  TableRow,
} from "@tremor/react";
import { Button } from "@tremor/react";
import { Minus } from "lucide-react";
import { Plus } from "lucide-react";
import { RevokeKeyButton } from "./revoke-key";

import { useState } from "react";
import { toast } from "sonner";
import { createKey } from "./action";

export default function Client({ keys }: { keys: any[] }) {
  const [isLoading, setLoading] = useState(false);
  const [newKey, setNewKey] = useState<string | undefined>(undefined);

  const curl = newKey
    ? `curl --location 'http:/localhost:3000/api/openai' \\
  --header 'Content-Type: application/json' \\
  --header 'Authorization: Bearer ${newKey}' \\
  --data '{"prompt": "unkey" }'
  `
    : null;

  return (
    <div>
      <Table>
        <TableHead>
          <TableRow>
            <TableHeaderCell>ID</TableHeaderCell>
            <TableHeaderCell>Key</TableHeaderCell>
            <TableHeaderCell>Remaining</TableHeaderCell>
            <TableHeaderCell className="text-right">Created At</TableHeaderCell>
            <TableHeaderCell>Revoke</TableHeaderCell>
          </TableRow>
        </TableHead>
        <TableBody>
          {keys.map((key) => (
            <TableRow key={key.id}>
              <TableCell className="font-mono">{key.id}</TableCell>
              <TableCell>{key.start}...</TableCell>
              <TableCell>{key.remaining ?? <Minus className="w-4 h-4" />}</TableCell>
              <TableCell className="text-right">{new Date(key.createdAt).toDateString()}</TableCell>
              <TableCell>{<RevokeKeyButton keyId={key.id} />}</TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>

      {curl ? (
        <Card>
          <pre className="font-mono">{curl}</pre>
        </Card>
      ) : null}
      <div className="mt-4 flex items-center justify-end">
        <form
          action={async () => {
            try {
              setLoading(true);
              const key = await createKey();
              setNewKey(key);
              toast("Copied key to clipboard.", {
                description: key,
              });
              navigator.clipboard.writeText(key as string);
            } finally {
              setLoading(false);
            }
          }}
        >
          <Button
            icon={Plus}
            loading={isLoading}
            disabled={isLoading}
            variant="secondary"
            type="submit"
          >
            Create new key
          </Button>
        </form>
      </div>
    </div>
  );
}
