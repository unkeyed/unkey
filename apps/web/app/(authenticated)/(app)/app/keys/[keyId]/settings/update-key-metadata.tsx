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
import { FormField } from "@/components/ui/form";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { toast } from "@/components/ui/toaster";
import { trpc } from "@/lib/trpc/client";
import { zodResolver } from "@hookform/resolvers/zod";
import { useRouter } from "next/navigation";
import { Form, useForm } from "react-hook-form";
import { z } from "zod";

const formSchema = z.object({
  keyId: z.string(),
  metadata: z.string(),
});
type Props = {
  apiKey: {
    id: string;
    meta: string | null;
  };
};

export const UpdateKeyMetadata: React.FC<Props> = ({ apiKey }) => {
  const router = useRouter();
  const [content, setContent] = useState<string>(apiKey.meta ?? "");
  const rows = Math.max(3, content.split("\n").length);
  const [isLoading, _setIsLoading] = useState(false);

  const form = useForm<z.infer<typeof formSchema>>({
    resolver: zodResolver(formSchema),
  });

  const updateMetadata = trpc.keySettings.updateMetadata.useMutation({
    onSuccess() {
      toast.success("Your remaining uses has been updated!");
      router.refresh();
    },
    onError(err) {
      console.error(err);
      toast.error(err.message);
    },
  });

  async function onSubmit(values: z.infer<typeof formSchema>) {
    updateMetadata.mutate(values);
  }
  return (
    <Form {...form}>
      <form onSubmit={form.handleSubmit(onSubmit)}>
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
              <FormField
                control={form.control}
                name="metadata"
                render={({ field }) => (
                  <Textarea
                    rows={rows}
                    {...field}
                    value={content}
                    onChange={(e) => setContent(e.target.value)}
                    className="w-full"
                    defaultValue={apiKey.meta ?? ""}
                    autoComplete="off"
                  />
                )}
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
    </Form>
  );
};
