"use client";

import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { useState } from "react";
import { create } from "./create";

const UnkeyElements = () => {
  const [key, setKey] = useState<string>("");
  const [secret, setSecret] = useState<string>("");
  async function onCreate(formData: FormData) {
    const res = await create(formData);
    if (res) {
      setKey(res.key?.key ?? "");
    }
  }
  const getData = async () => {
    const res = await fetch("/api/secret", {
      headers: {
        Authorization: `Bearer ${key}`,
      },
    });
    const data = await res.json();
    setSecret(data.result);
  };
  return (
    <div className="mt-8">
      <Card className="w-[350px]">
        <CardHeader>
          <CardTitle>Create API Key</CardTitle>
          <CardDescription>Create your API key so you can interact with our API.</CardDescription>
        </CardHeader>
        <form action={onCreate}>
          <CardContent>
            <div className="grid items-center w-full gap-4">
              <div className="flex flex-col space-y-1.5">
                <Label htmlFor="name">API Key Name</Label>
                <Input name="name" placeholder="My Awesome API " />
              </div>
            </div>
          </CardContent>
          <CardFooter className="flex justify-between">
            <Button type="submit">Create Key</Button>
          </CardFooter>
        </form>
      </Card>
      {key && key.length > 0 && (
        <>
          <Card className="w-[350px] mt-8">
            <CardHeader>
              <CardTitle>API Key</CardTitle>
              <CardDescription>Here is your API key. Keep it safe!</CardDescription>
            </CardHeader>
            <CardContent>
              <div className="grid items-center w-full gap-4">
                <div className="flex flex-col space-y-1.5">
                  <Label htmlFor="name">API Key</Label>
                  <Input name="name" value={key} />
                </div>
              </div>
            </CardContent>
          </Card>
          <Card className="w-[350px] mt-8">
            <CardHeader>
              <CardTitle>Get Secret Data </CardTitle>
              <CardDescription>Retrieve secret data from API </CardDescription>
            </CardHeader>
            <CardContent>
              <Button onClick={getData} variant="outline">
                Get Data
              </Button>
              <div className="grid items-center w-full gap-4">
                <div className="flex flex-col space-y-1.5">
                  <Label htmlFor="name">Secret Data</Label>
                  <Input name="name" value={JSON.stringify(secret)} />
                </div>
              </div>
            </CardContent>
          </Card>
        </>
      )}
    </div>
  );
};

export { UnkeyElements };
