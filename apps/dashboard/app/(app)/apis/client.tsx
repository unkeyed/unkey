"use client";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { PostHogIdentify } from "@/providers/PostHogProvider";
import { useUser } from "@clerk/nextjs";
import { Empty } from "@unkey/ui";
import { Button } from "@unkey/ui";
import { BookOpen, Search } from "lucide-react";
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

  const [localData, setLocalData] = useState(apis);

  if (isLoaded && user) {
    PostHogIdentify({ user });
  }

  useEffect(() => {
    if (apis.length) {
      setLocalData(apis);
    }
  }, [apis]);

  return (
    <div>
      <section className="mb-4 flex flex-col gap-4 md:flex-row md:items-center">
        <div className="border-border focus-within:border-primary/40 flex h-8 flex-grow items-center gap-2 rounded-md border bg-background px-3 py-2 text-sm">
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
      </section>
      {apis.length ? (
        <ul className="grid grid-cols-1 gap-x-6 gap-y-8 lg:grid-cols-2 xl:grid-cols-3">
          {localData.map((api) => (
            <Link key={api.id} href={`/apis/${api.id}`}>
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
        <Empty>
          <Empty.Icon />
          <Empty.Title>No APIs found</Empty.Title>
          <Empty.Description>
            You haven&apos;t created any APIs yet. Create one to get started.
          </Empty.Description>
          <Empty.Actions>
            <CreateApiButton key="createApi" />
            <Link href="/docs" target="_blank">
              <Button>
                <BookOpen />
                Read the docs
              </Button>
            </Link>
          </Empty.Actions>
        </Empty>
      )}
    </div>
  );
}
