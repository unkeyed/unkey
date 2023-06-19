"use client";
import { PageHeader } from "@/components/PageHeader";
import { Policy } from "@unkey/policies";
import { Checkbox } from "@/components/ui/checkbox";
import { Label } from "@/components/ui/label";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { useToast } from "@/components/ui/use-toast";
import { DeleteKeyButton } from "../DeleteKey";
import { Badge } from "@/components/ui/badge";
import { Trash } from "lucide-react";
import { Button } from "@/components/ui/button";

const _allActions = ["create", "read", "update", "delete"];

type Props = {
  apiKey: {
    id: string;
    start: string;
    createdAt: Date;
  };
};
export const Client: React.FC<Props> = ({ apiKey }) => {
  const { toast } = useToast();

  // const policy = apiKey.policy ? Policy.parse(apiKey.policy) : null;

  return (
    <div className="px-4 mx-auto mt-8 max-w-7xl sm:px-6 lg:px-8">
      <PageHeader
        title={apiKey.id}
        description={`created at ${apiKey.createdAt.toUTCString()}`}
        actions={[
          <Badge key="key">{apiKey.start}...</Badge>,
          <DeleteKeyButton key="delete" keyId={apiKey.id}>
            <Button variant="secondary">
              <Trash className="w-4 h-4 mr-2" />
              <span>Revoke</span>
            </Button>
          </DeleteKeyButton>,
        ]}
      />
      <div className="mt-8 space-y-10 divide-y divide-zinc-900/10" />
    </div>
  );
};
