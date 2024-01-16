"use client";
import { Loading } from "@/components/dashboard/loading";
import { PageHeader } from "@/components/dashboard/page-header";
import { Button } from "@/components/ui/button";
import { Label } from "@/components/ui/label";
import { ScrollArea } from "@/components/ui/scroll-area";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { toast } from "@/components/ui/toaster";
import { trpc } from "@/lib/trpc/client";
import { Api, VercelBinding } from "@unkey/db";
import { X } from "lucide-react";
import { useRouter } from "next/navigation";
import { useState } from "react";
import { WorkspaceSwitcher } from "./workspace";

type Props = {
  projects: { id: string; name: string }[];
  apis: Api[];
  returnUrl: string;
  integrationId: string;
  accessToken: string;
  vercelTeamId: string | null;
};

export const Client: React.FC<Props> = ({
  projects,
  apis,
  returnUrl,
  integrationId,
  accessToken,
  vercelTeamId,
}) => {
  const [projectId, setProjectId] = useState<string | undefined>(
    projects.length === 1 ? projects[0].id : undefined,
  );

  const [selectedApis, setSelectedApis] = useState<
    Record<VercelBinding["environment"], string | null>
  >({
    production: null,
    preview: null,
    development: null,
  });
  const router = useRouter();

  const disabled =
    !projectId || !(selectedApis.development || selectedApis.preview || selectedApis.production);

  const create = trpc.vercel.setupProject.useMutation({
    onSuccess: () => {
      toast.success("Successfully added environment variables to your Vercel project");

      toast("Redirecting back to Vercel");
      router.push(returnUrl);
    },
    onError: (err) => {
      toast.error(err.message);
    },
  });

  return (
    <div className="container min-h-screen mx-auto mt-8">
      <PageHeader
        title="Connect Vercel Project"
        description="You can add more projects later"
        actions={[<WorkspaceSwitcher />]}
      />

      <div className="flex flex-col flex-1 flex-grow gap-16">
        <div className="flex flex-col gap-2">
          <Label>Vercel Project</Label>
          <Select
            value={projectId}
            onValueChange={(id) => {
              setProjectId(id);
            }}
          >
            <SelectTrigger className="w-full">
              <SelectValue
                defaultValue={projectId}
                placeholder="Select a connected project from Vercel"
              />
            </SelectTrigger>
            <SelectContent>
              <ScrollArea className="max-h-64">
                {projects.map((p) => (
                  <SelectItem key={p.id} value={p.id}>
                    {p.name}
                  </SelectItem>
                ))}
              </ScrollArea>
            </SelectContent>
          </Select>
        </div>

        {projectId ? (
          <div>
            <p className="text-sm text-content-subtle">
              Connect your existign Unkey APIs to your project's environments. We suggest using
              different APIs per environment for better isolation.
            </p>
            <div className="flex flex-col gap-4 mt-4">
              {Object.entries(selectedApis).map(([environment, apiId]) => (
                <div key={environment + apiId}>
                  <Label className="capitalize md:w-auto shrink-0 whitespace-nowrap">
                    {environment}
                  </Label>

                  <div className="flex items-center gap-2 mt-2">
                    <Select
                      onValueChange={(id) => {
                        setSelectedApis({ ...selectedApis, [environment]: id });
                      }}
                      defaultValue={apiId ?? undefined}
                    >
                      <SelectTrigger className="w-full">
                        <SelectValue />
                      </SelectTrigger>
                      <SelectContent>
                        <ScrollArea className="max-h-64">
                          {apis.map((api) => (
                            <SelectItem key={api.id} value={api.id}>
                              {api.name}
                            </SelectItem>
                          ))}
                        </ScrollArea>
                      </SelectContent>
                    </Select>
                    <Button
                      variant="ghost"
                      size="icon"
                      onClick={() => {
                        setSelectedApis({ ...selectedApis, [environment]: null });
                      }}
                    >
                      <X className="w-4 h-4" />
                    </Button>
                  </div>
                </div>
              ))}
            </div>
          </div>
        ) : null}
      </div>

      <footer className="flex items-center justify-end gap-4 mt-8">
        <Button
          variant="secondary"
          onClick={() => {
            window.close();
          }}
        >
          Cancel
        </Button>
        <Button
          disabled={disabled}
          variant={disabled ? "disabled" : "primary"}
          onClick={() => {
            create.mutate({
              projectId: projectId!,
              integrationId,
              accessToken,
              vercelTeamId: vercelTeamId,
              apiIds: {
                production: selectedApis.production,
                preview: selectedApis.preview,
                development: selectedApis.development,
              },
            });
          }}
        >
          {create.isLoading ? <Loading /> : "Save"}
        </Button>
      </footer>
    </div>
  );
};
