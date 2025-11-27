/**
 * Deprecated with new auth
 * Hiding for now until we decide if we want to fix it up or toss it
 */

"use client";;
import { PageHeader } from "@/components/dashboard/page-header";
import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Label } from "@/components/ui/label";
import { useTRPC } from "@/lib/trpc/client";
import { cn } from "@/lib/utils";
import type { Api, Key, VercelBinding } from "@unkey/db";
import { Dots, ExternalLink, Link4, Plus, Refresh3, Trash, Unlink } from "@unkey/icons";
import {
  Button,
  Empty,
  Input,
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
  Tooltip,
  TooltipContent,
  TooltipTrigger,
  toast,
} from "@unkey/ui";
import { formatDistanceToNow } from "date-fns";
import Link from "next/link";
import { useRouter } from "next/navigation";
import type React from "react";
import { useState } from "react";

import { useMutation } from "@tanstack/react-query";

type Props = {
  integration: {
    id: string;
  };
  apis: Record<string, Api>;
  rootKeys: Record<string, Key>;
  projects: {
    id: string;
    name: string;
    bindings: Record<
      VercelBinding["environment"],
      Record<
        VercelBinding["resourceType"],
        | (VercelBinding & {
          updatedBy: {
            id: string;
            name: string;
            image: string | null;
          };
        })
        | null
      >
    >;
  }[];
};

export const Client: React.FC<Props> = ({ projects, integration, apis, rootKeys }) => {
  projects.sort((a, b) => a.name.localeCompare(b.name));

  if (projects.length === 0) {
    return (
      <Empty>
        <Empty.Icon />
        <Empty.Title>No connected projects found</Empty.Title>
        <Empty.Description>Connect a Vercel project now</Empty.Description>
        <Empty.Actions>
          <Link href="https://vercel.com/integrations/unkey" target="_blank">
            <Button variant="ghost">Vercel Integration</Button>
          </Link>
        </Empty.Actions>
      </Empty>
    );
  }

  const environments: Record<VercelBinding["environment"], string> = {
    production: "Production",
    preview: "Preview",
    development: "Development",
  };

  return (
    <>
      <PageHeader
        title="Connected Projects"
        description="Connect a Vercel project to an API to automatically add the necessary environment variables to your project."
        actions={[
          <Link
            key="vercelIntegration"
            href={`https://vercel.com/dashboard/integrations/${integration.id}`}
            target="_blank"
          >
            <Button>Configure Vercel</Button>
          </Link>,
        ]}
      />
      <div className="flex items-center justify-center w-full ">
        <ul className="w-full space-y-8">
          {projects.map((project) => {
            return (
              <li key={project.id}>
                <div className="flex items-center justify-between gap-2">
                  <h3 className="flex items-center gap-2">
                    <svg
                      className="w-4 h-4"
                      viewBox="0 0 76 65"
                      fill="none"
                      xmlns="http://www.w3.org/2000/svg"
                    >
                      <path d="M37.5274 0L75.0548 65H0L37.5274 0Z" fill="#000000" />
                    </svg>
                    <span className="font-semibold">{project.name}</span>
                  </h3>

                  <Button variant="ghost" shape="square">
                    <Dots iconSize="md-medium" />
                  </Button>
                </div>

                <ul className="w-full mt-2 overflow-hidden border divide-y rounded">
                  {Object.entries(environments).map(([e, envLabel]) => {
                    const environment = e as VercelBinding["environment"];
                    const binding = project.bindings[environment];

                    return (
                      <li key={environment}>
                        <div
                          className={cn(
                            "flex flex-col items-center justify-between gap-8 p-4 md:flex-row hover:bg-white ",
                            {
                              "bg-white": binding,
                              "opacity-50 bg-background-subtle hover:opacity-100 ": !binding,
                            },
                          )}
                        >
                          <div className="flex items-center w-full md:w-1/5">
                            {binding ? (
                              <Link4 iconSize="md-thin" className="rotate-90" />
                            ) : (
                              <Unlink iconSize="md-thin" className="rotate-90" />
                            )}
                            <span className="text-xs text-content">{envLabel}</span>
                          </div>

                          <div className="flex justify-end w-full md:w-2/5">
                            <ConnectedResource
                              type="API"
                              binding={binding?.apiId}
                              rootKeys={rootKeys}
                              apis={apis}
                              integrationId={integration.id}
                              projectId={project.id}
                              environment={environment}
                            />
                          </div>
                          <div className="flex justify-end w-full md:w-2/5">
                            <ConnectedResource
                              type="Root Key ID"
                              binding={binding?.rootKey}
                              rootKeys={rootKeys}
                              apis={apis}
                              integrationId={integration.id}
                              projectId={project.id}
                              environment={environment}
                            />
                          </div>
                        </div>
                      </li>
                    );
                  })}
                </ul>
              </li>
            );
          })}
        </ul>
      </div>
    </>
  );
};

const ConnectedResource: React.FC<{
  type: "API" | "Root Key ID";
  projectId: string;
  integrationId: string;
  environment: VercelBinding["environment"];
  binding:
  | (VercelBinding & {
    updatedBy: {
      id: string;
      name: string;
      image: string | null;
    };
  })
  | null;
  apis: Record<string, Api>;
  rootKeys: Record<string, Key>;
}> = (props) => {
  const trpc = useTRPC();
  const router = useRouter();
  const [selectedResourceId, setSelectedResourceId] = useState(props.binding?.resourceId);

  const updateApiId = useMutation(trpc.vercel.upsertApiId.mutationOptions({
    onSuccess: () => {
      router.refresh();
      toast.success("Updated the environment variable in Vercel");
    },
    onError(err) {
      console.error(err);
      toast.error(err.message);
    },
  }));

  const rerollRootKey = useMutation(trpc.vercel.upsertNewRootKey.mutationOptions({
    onSuccess: () => {
      router.refresh();
      toast.success(
        "Successfully rolled your root key and updated the environment variable in Vercel",
      );
    },
    onError(err) {
      console.error(err);
      toast.error(err.message);
    },
  }));
  const unbind = useMutation(trpc.vercel.unbind.mutationOptions({
    onSuccess: () => {
      router.refresh();
      toast.success(`Successfully unbound ${props.type} from Vercel`);
    },
    onError(err) {
      console.error(err);
      toast.error(err.message);
    },
  }));

  const isLoading = updateApiId.isPending || rerollRootKey.isPending || unbind.isPending;

  return (
    <div className="flex items-center w-full gap-2 ">
      <Label className="w-1/5 md:w-auto shrink-0 whitespace-nowrap">{props.type}</Label>
      {props.type === "API" ? (
        <Select
          value={selectedResourceId}
          onValueChange={(id) => {
            setSelectedResourceId(id);
            updateApiId.mutate({
              apiId: id,
              projectId: props.projectId,
              integrationId: props.integrationId,
              environment: props.environment,
            });
          }}
        >
          <SelectTrigger className="w-full">
            <SelectValue defaultValue={selectedResourceId ?? "None"} />
          </SelectTrigger>
          <SelectContent>
            {Object.values(props.apis).map((api) => (
              <SelectItem key={api.id} value={api.id}>
                {api.name}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      ) : (
        <Tooltip>
          <TooltipTrigger className="w-full">
            <Input disabled value={props.binding?.resourceId} />
          </TooltipTrigger>
          <TooltipContent>
            Because we don't store the root key itself, you can not select a different existing key.
            <br />
            Use the button on the right to generate a new key and update the environment variable in
            Vercel.
          </TooltipContent>
        </Tooltip>
      )}

      <DropdownMenu>
        <DropdownMenuTrigger>
          <Button variant="ghost" shape="square" loading={isLoading}>
            <Dots iconSize="md-medium" />
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent>
          {props.binding ? (
            <>
              <DropdownMenuLabel className="flex items-center justify-between w-full gap-2">
                <span className="text-sm text-content">
                  Edited{" "}
                  {props.binding.updatedAtM
                    ? formatDistanceToNow(new Date(props.binding.updatedAtM), {
                      addSuffix: true,
                    })
                    : "recently"}{" "}
                  by {props.binding.updatedBy.name}
                </span>
                <Avatar className="w-6 h-6 ">
                  {/* <AvatarImage
                    src={props.binding?.updatedBy.image}
                    alt={props.binding?.updatedBy.name}
                  /> */}
                  {props.binding?.updatedBy.image && (
                    <AvatarImage
                      src={props.binding?.updatedBy.image}
                      alt={props.binding?.updatedBy.name}
                    />
                  )}
                  <AvatarFallback className="w-6 h-6 lg:w-5 lg:h-5 bg-gray-100 border border-gray-500 rounded-md">
                    {(props.binding?.updatedBy.name ?? "U").slice(0, 2).toUpperCase()}
                  </AvatarFallback>
                </Avatar>
              </DropdownMenuLabel>
              <DropdownMenuSeparator />
            </>
          ) : props.type === "API" ? (
            <DropdownMenuLabel>Select an API to bind</DropdownMenuLabel>
          ) : null}

          {props.binding ? (
            <DropdownMenuItem>
              <Link
                className="flex items-center"
                href={
                  props.binding.resourceType === "apiId"
                    ? `/${props.binding.workspaceId}/api/${props.binding.resourceId}`
                    : `/${props.binding.workspaceId}/settings/root-keys/${props.binding.resourceId}`
                }
              >
                <ExternalLink className="w-4 h-4 mr-2" />
                Go to {props.binding.resourceType === "apiId" ? "API" : "Root Key ID"}
              </Link>
            </DropdownMenuItem>
          ) : null}

          {props.type === "Root Key ID" ? (
            <DropdownMenuItem
              disabled={unbind.isPending}
              onClick={() => {
                rerollRootKey.mutate({
                  integrationId: props.integrationId,
                  projectId: props.projectId,
                  environment: props.environment,
                });
              }}
            >
              <Tooltip>
                <TooltipTrigger className="flex items-center gap-2">
                  {props.binding ? (
                    <>
                      <Refresh3 iconSize="md-medium" className="transform scale-x-[-1]" />
                      Reroll the Key
                    </>
                  ) : (
                    <>
                      <Plus iconSize="md-medium" />
                      Generate new Key
                    </>
                  )}
                </TooltipTrigger>
                <TooltipContent>
                  This will generate a new key and update the environment variable in Vercel.
                </TooltipContent>
              </Tooltip>
            </DropdownMenuItem>
          ) : null}

          {props.binding ? (
            <DropdownMenuItem
              onClick={() => {
                if (!props.binding) {
                  toast.error("Unable to unbind. Please refresh and try again.");
                  return;
                }
                unbind.mutate({
                  bindingId: props.binding.id,
                });
              }}
              className="flex items-center gap-2"
            >
              <Trash className="w-4 h-4" />
              Remove the environment variable from Vercel
            </DropdownMenuItem>
          ) : null}
        </DropdownMenuContent>
      </DropdownMenu>
    </div>
  );
};
