"use client";
import { Button } from "@/components/ui/button";
import React, { useState } from "react";

import { SubmitButton } from "@/components/dashboard/submit-button";
import {
  Card,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { toast } from "@/components/ui/toaster";
import { updateMetadata } from "./actions";
type Props = {
  apiKey: {
    id: string;
    meta: string | null;
  };
};

export const UpdateKeyMetadata: React.FC<Props> = ({ apiKey }) => {
  const [content, setContent] = useState<string>(apiKey.meta ?? "");
  const rows = Math.max(3, content.split("\n").length);
  return (
    <form
      action={async (formData: FormData) => {
        const res = await updateMetadata(formData);
        if (res.error) {
          toast.error(res.error.message);
          return;
        }

        toast.success("Metadata updated");
      }}
    >
      <Card>
        <CardHeader>
          <CardTitle>Metadata</CardTitle>
          <CardDescription>
            Store json, or any other data you want to associate with this key. Whenever you verify
            this key, we'll return the metadata to you.
          </CardDescription>
        </CardHeader>
        <CardContent className="flex justify-between item-center">
          <div className="flex flex-col w-full space-y-2">
            <input type="hidden" name="keyId" value={apiKey.id} />

            <Label htmlFor="metadata">Metadata</Label>
            <Textarea
              rows={rows}
              value={content}
              onChange={(e) => setContent(e.target.value)}
              name="metadata"
              className="w-full"
              defaultValue={apiKey.meta ?? ""}
              autoComplete="off"
            />
          </div>
        </CardContent>
        <CardFooter className="justify-end gap-4">
          <Button
            variant="secondary"
            type="button"
            onClick={() => {
              try {
                const parsed = JSON.parse(content);
                setContent(JSON.stringify(parsed, null, 2));
              } catch (e) {
                toast.error((e as Error).message);
              }
            }}
          >
            Format Json
          </Button>
          <SubmitButton label="Save" />
        </CardFooter>
      </Card>
    </form>
  );
};
