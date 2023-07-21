"use client";
import { EmptyPlaceholder } from "@/components/dashboard/empty-placeholder";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { CreateApiButton } from "@/components/dashboard/create-api";
import { Button } from "@/components/ui/button";
import { PageHeader } from "@/components/dashboard/page-header";
import { Separator } from "@/components/ui/separator";
import { useState } from "react";
import Link from "next/link";
import { Icons } from "@/components/ui/icons";

type ApiWithKeys = {
  id: string;
  name: string;
  keys: {
    count: number;
  }[];
}[];

export function ApiList({ apis }: { apis: ApiWithKeys }) {
  const [localData, setLocalData] = useState(apis);
  return (
    <div>
      <PageHeader title="Applications" description="Manage your APIs" />
      <Separator className="my-6" />
      <section className=" my-4 flex md:items-center gap-4 flex-col md:flex-row">
        <div className="flex h-10 flex-grow items-center gap-2 rounded-md border border-input bg-transparent px-3 py-2 text-sm focus-within:border-primary/40">
          <Icons.search size={18} />
          <input
            className="disabled:cursor-not-allowed bg-transparent flex-grow disabled:opacity-50 placeholder:text-muted-foreground focus-visible:outline-none  "
            placeholder="Search.."
            onChange={(e) => {
              const filtered = apis.filter((a) =>
                a.name.toLowerCase().includes(e.target.value.toLowerCase()),
              );
              setLocalData(filtered);
            }}
          />
        </div>
        <CreateApiButton key="createApi" />
      </section>
      {apis.length ? (
        <ul role="list" className="grid grid-cols-1 gap-x-6 gap-y-8 lg:grid-cols-3 xl:gap-x-8">
          {localData.map((api) => (
            <Link key={api.id} href={`/app/${api.id}`}>
              <Card className="duration-500 hover:border-primary/10 group hover:drop-shadow-md bg-gradient-to-tl dark:via-zinc-900/50 dark:from-zinc-900 dark:to-zinc-950 relative">
                <div
                  className="absolute left-0 top-0 h-px w-36 group-hover:w-[300px] transition-all duration-500"
                  style={{
                    background:
                      "linear-gradient(90deg, rgba(56, 189, 248, 0) 0%, rgba(56, 189, 248, 0) 0%, rgba(232, 232, 232, 0.2) 33.02%, rgba(143, 143, 143, 0.67) 64.41%, rgba(236, 72, 153, 0) 98.93%);",
                  }}
                />
                <div className=" absolute bottom-0 h-2 left-0 group-hover:w-80 transition-all duration-500 ease-in-out  w-20 blur-2xl opacity-30 bg-white" />
                <div className=" absolute delay-100 bottom-0 h-1 right-0 group-hover:w-80 transition-all duration-500 ease-in-out  w-20 blur-2xl opacity-40 bg-white" />
                <CardHeader>
                  <div className=" flex items-center justify-between">
                    <CardTitle>{api.name}</CardTitle>
                    <Icons.api size={18} />
                  </div>
                  <CardDescription>{api.id}</CardDescription>
                </CardHeader>
                <CardContent>
                  <dl className="text-sm leading-6 divide-y divide-gray-100 ">
                    <div className="flex justify-between py-3 gap-x-4">
                      <dt className="text-gray-500 dark:text-gray-400">API Keys</dt>
                      <dd className="flex items-start gap-x-2">
                        <div className="font-medium text-gray-900 dark:text-gray-200">
                          {api.keys.at(0)?.count ?? 0}
                        </div>
                      </dd>
                    </div>
                  </dl>
                </CardContent>
              </Card>
            </Link>
          ))}
        </ul>
      ) : (
        <EmptyPlaceholder className=" my-4">
          <EmptyPlaceholder.Icon />
          <EmptyPlaceholder.Title>No API Keys Yet</EmptyPlaceholder.Title>
          <EmptyPlaceholder.Description>
            You haven&apos;t created any API yet. Create one to get started.
          </EmptyPlaceholder.Description>
          <div className="gap-2 flex items-center justify-center flex-col md:flex-row">
            <CreateApiButton key="createApi" className="" />
            <Link href="/docs" target="_blank">
              <Button variant="outline" className=" gap-2 items-center w-full">
                <Icons.docs size={18} className=" w-4 h-4 md:w-5 md:h-5" />
                Read the docs
              </Button>
            </Link>
          </div>
        </EmptyPlaceholder>
      )}
    </div>
  );
}
