"use client";
import { PageHeader } from "@/components/PageHeader";
import { Policy } from "@unkey/policies";
import { Checkbox } from "@/components/ui/checkbox";
import { Label } from "@/components/ui/label";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { useToast } from "@/hooks/use-toast";
import { DeleteKeyButton } from "../DeleteKey";
import { Badge } from "@/components/ui/badge";
import { Trash } from "lucide-react";
import { Button } from "@/components/ui/button";

const allActions = ["create", "read", "update", "delete"];

type Props = {
  apiKey: {
    id: string;
    start: string;
    createdAt: Date;
    policy: string | null;
  };
};
export const Client: React.FC<Props> = ({ apiKey }) => {
  const { toast } = useToast();

  const policy = apiKey.policy ? Policy.parse(apiKey.policy) : null;

  return (
    <div className="px-4 mx-auto mt-8 max-w-7xl sm:px-6 lg:px-8">
      <PageHeader
        title={apiKey.id}
        description={`created at ${apiKey.createdAt.toUTCString()}`}
        actions={[
          <Badge key="key">{apiKey.start}...</Badge>,
          <DeleteKeyButton key="delte" keyId={apiKey.id}>
            <Button variant="secondary">
              <Trash className="w-4 h-4 mr-2" />
              <span>Revoke</span>
            </Button>
          </DeleteKeyButton>,
        ]}
      />
      <div className="mt-8 space-y-10 divide-y divide-zinc-900/10">
        {policy?.statements.map((statement, _i) => {
          return Object.entries(statement.resources).map(([_resourceType, resources]) => (
            <div className="flex flex-col md:flex-row ">
              {/* <div className="w-full md:w-1/5">
    This is here in preparateion for the future where we'll have more than just channels
                                <CardHeader>
                                    <CardTitle>{resourceType}</CardTitle>
                                </CardHeader>

                            </div> */}
              <Card className="w-full ">
                <CardContent>
                  {Object.entries(resources ?? {}).map(([grid, permissions]) => {
                    return (
                      <div className="flex flex-col items-start justify-between w-full py-6  md:flex-row md:items-center gap-4">
                        <span className="text-sm font-medium text-white leading-6">{grid}</span>
                        <div className="flex flex-wrap items-center text-sm text-zinc-400 justify-right gap-4">
                          {allActions.map((action) => (
                            <div key={action} className="flex items-center space-x-2">
                              <Checkbox
                                checked={permissions.includes(action)}
                                onClick={() => {
                                  toast({
                                    title: "You can't update permissions yet",
                                    description: "Coming soon...",
                                  });
                                }}
                              />

                              <Label>{action}</Label>
                            </div>
                          ))}
                        </div>
                      </div>
                    );
                  })}
                </CardContent>
              </Card>
            </div>
          ));
        })}
      </div>
    </div>
  );
};
