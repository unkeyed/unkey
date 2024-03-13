"use client";
import { EmptyPlaceholder } from "@/components/dashboard/empty-placeholder";
import { PageHeader } from "@/components/dashboard/page-header";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Separator } from "@/components/ui/separator";
import { PostHogIdentify } from "@/providers/PostHogProvider";
import { useUser } from "@clerk/nextjs";
import { BookOpen, Code, Search } from "lucide-react";
import Link from "next/link";
import { useEffect, useState } from "react";
import { CreateApiButton } from "./create-api-button";
type ApiWithKeys = {
  id: string;
  name: string;
  keys: {
    count: number;
  }[];
}[];

export function ApiList({ apis }: { apis: ApiWithKeys }) {
  const { user, isLoaded } = useUser();
  useEffect(() => {
    if (apis.length) {
      setLocalData(apis);
    }
  }, [apis]);
  const [localData, setLocalData] = useState(apis);
  if (isLoaded && user) {
    PostHogIdentify({ user });
  }
  return (
    <div>
      <PageHeader title="Applications" description="Manage your APIs" />
      <Separator className="my-6" />
      <section className="my-4 flex flex-col gap-4 md:flex-row md:items-center">
        <div className="border-border focus-within:border-primary/40 flex h-8 flex-grow items-center gap-2 rounded-md border bg-transparent px-3 py-2 text-sm">
          <Search className="h-4 w-4" />
          <input
            className="placeholder:text-content-subtle flex-grow bg-transparent focus-visible:outline-none disabled:cursor-not-allowed disabled:opacity-50 "
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
        <ul className="grid grid-cols-1 gap-x-6 gap-y-8 lg:grid-cols-2 xl:grid-cols-3">
          {localData.map((api) => (
            <Link key={api.id} href={`/app/apis/${api.id}`}>
              <Card className="hover:border-primary/50 group relative overflow-hidden duration-500 ">
                <CardHeader>
                  <div className="flex items-center justify-between">
                    <CardTitle className="truncate">{api.name}</CardTitle>
                  </div>
                  <CardDescription>{api.id}</CardDescription>
                </CardHeader>
                <CardContent>
                  <dl className="divide-y divide-gray-100 text-sm leading-6 ">
                    <div className="flex justify-between gap-x-4 py-3">
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
        <EmptyPlaceholder className="my-4 ">
          <EmptyPlaceholder.Icon>
            <Code />
          </EmptyPlaceholder.Icon>
          <EmptyPlaceholder.Title>No APIs found</EmptyPlaceholder.Title>
          <EmptyPlaceholder.Description>
            You haven&apos;t created any APIs yet. Create one to get started.
          </EmptyPlaceholder.Description>
          <div className="flex flex-col items-center justify-center gap-2 md:flex-row">
            <CreateApiButton key="createApi" className="" />
            <Link href="/docs" target="_blank">
              <Button variant="secondary" className="w-full items-center gap-2 ">
                <BookOpen className="h-4 w-4 md:h-5 md:w-5" />
                Read the docs
              </Button>
            </Link>
          </div>
        </EmptyPlaceholder>
      )}
    </div>
  );
}
