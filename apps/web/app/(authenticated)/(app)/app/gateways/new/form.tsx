"use client";
import { Loading } from "@/components/dashboard/loading";
import { Accordion, AccordionContent } from "@/components/ui/accordion";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import {
  Form,
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from "@/components/ui/form";
import { Input } from "@/components/ui/input";
import { Separator } from "@/components/ui/separator";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { toast } from "@/components/ui/toaster";
import { trpc } from "@/lib/trpc/client";
import { zodResolver } from "@hookform/resolvers/zod";
import * as AccordionPrimitive from "@radix-ui/react-accordion";
import { ChevronDown, Eye, EyeOff, X } from "lucide-react";
import { useRouter } from "next/navigation";
import { useState } from "react";
import { useFieldArray, useForm } from "react-hook-form";
import { z } from "zod";

const formSchema = z.object({
  subdomain: z.string().regex(/^[a-zA-Z0-9-]+$/),
  origin: z.string().url(),
  headerRewrites: z.array(
    z.object({
      name: z.string().min(1),
      value: z.string().min(1),
      show: z.boolean().default(false).optional(),
    }),
  ),
});
export const CreateGatewayForm: React.FC = () => {
  const router = useRouter();
  const form = useForm<z.infer<typeof formSchema>>({
    resolver: zodResolver(formSchema),
    mode: "all",
    shouldFocusError: true,
  });

  const fields = useFieldArray({ control: form.control, name: "headerRewrites" });
  const create = trpc.gateway.create.useMutation({
    onSuccess(_, variables) {
      toast.success("Gateway Created", {
        description: "Your Gateway has been created",
        duration: 10_000,
        action: {
          label: "Go to",
          onClick: () => {
            router.push(`https://${variables.subdomain}.unkey.io`);
          },
        },
      });
    },
    onError(err) {
      toast.error("An error occured", {
        description: err.message,
      });
    },
  });

  async function onSubmit(values: z.infer<typeof formSchema>) {
    create.mutate(values);
  }

  // const snippet = `curl -XPOST '${process.env.NEXT_PUBLIC_UNKEY_API_URL ?? "https://api.unkey.dev"}/v1/keys.verifyKey' \\
  // -H 'Content-Type: application/json' \\
  // -d '{
  //   "key": "${key.data?.key}"
  // }'`;

  const [newHeaderRewriteName, setNewHeadeRewriteName] = useState("");
  const [newHeaderRewriteValue, setNewHeadeRewriteValue] = useState("");

  return (
    <div>
      <Card>
        <CardHeader>
          <CardTitle>Configure Gateway</CardTitle>
        </CardHeader>
        <CardContent>
          <Separator className="mb-4" />
          <Form {...form}>
            <form className="flex flex-col gap-8" onSubmit={form.handleSubmit(onSubmit)}>
              <FormField
                control={form.control}
                name="subdomain"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>Domain</FormLabel>
                    <FormControl>
                      <div className="flex mt-2 border rounded-md shadow-sm focus-within:border-primary">
                        <span className="inline-flex items-center px-3 text-content bg-background-subtle rounded-l-md sm:text-sm ">
                          https://
                        </span>
                        <input
                          className="flex-1 block w-full h-8 min-w-0 px-3 py-2 text-sm bg-transparent placeholder:text-content-subtle focus-visible:outline-none disabled:cursor-not-allowed disabled:opacity-50 "
                          // className="flex w-full h-8 px-3 py-2 text-sm bg-transparent border rounded-md border-border focus:border-primary file:border-0 file:bg-transparent file:text-sm file:font-medium placeholder:text-content-subtle focus-visible:outline-none disabled:cursor-not-allowed disabled:opacity-50"
                          {...field}
                        />
                        <span className="inline-flex items-center px-3 text-content bg-background-subtle rounded-r-md sm:text-sm">
                          .unkey.io
                        </span>
                      </div>
                    </FormControl>
                    <FormDescription>
                      The domain where your gateway will be available
                    </FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />
              <FormField
                control={form.control}
                name="origin"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>Origin</FormLabel>
                    <FormControl>
                      <Input {...field} />
                    </FormControl>
                    <FormDescription>The origin host</FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />

              <Separator />

              <Accordion
                type="single"
                collapsible
                className="w-full p-4 border rounded-lg bg-background border-border"
              >
                <AccordionPrimitive.Item value="item-1">
                  <AccordionPrimitive.Header className="flex">
                    <AccordionPrimitive.Trigger className="flex flex-1 items-center justify-start text-xs text-content gap-2 transition-all [&[data-state=open]>svg]:rotate-180">
                      <ChevronDown className="w-4 h-4 transition-transform duration-200 shrink-0" />
                      Header Rewrites
                    </AccordionPrimitive.Trigger>
                  </AccordionPrimitive.Header>

                  <AccordionContent>
                    <div className="flex items-start justify-between gap-4 mt-4">
                      <div className="w-full">
                        <Input
                          value={newHeaderRewriteName}
                          onChange={(v) => setNewHeadeRewriteName(v.currentTarget.value)}
                        />

                        <FormDescription>Name</FormDescription>
                        <FormMessage />
                      </div>
                      <div className="w-full">
                        <Input
                          value={newHeaderRewriteValue}
                          type="password"
                          onChange={(v) => setNewHeadeRewriteValue(v.currentTarget.value)}
                        />
                        <FormDescription>Value (encrypted at rest)</FormDescription>
                        <FormMessage />
                      </div>

                      <Button
                        type="button"
                        variant={
                          newHeaderRewriteName.length === 0 || newHeaderRewriteValue.length === 0
                            ? "disabled"
                            : "primary"
                        }
                        disabled={
                          newHeaderRewriteName.length === 0 || newHeaderRewriteValue.length === 0
                        }
                        onClick={() => {
                          fields.append({
                            value: newHeaderRewriteName,
                            name: newHeaderRewriteValue,
                          });
                          setNewHeadeRewriteName("");
                          setNewHeadeRewriteValue("");
                        }}
                      >
                        Add
                      </Button>
                    </div>

                    {fields.fields.length > 0 ? (
                      <Table className="mt-8 no-scrollbar">
                        <TableHeader>
                          <TableRow>
                            <TableHead>Name</TableHead>
                            <TableHead>Value</TableHead>

                            <TableHead className="w-20" />
                          </TableRow>
                        </TableHeader>
                        <TableBody>
                          {fields.fields.map((f, index) => (
                            <TableRow key={f.id}>
                              <TableCell className="flex-grow pl-0">
                                <FormField
                                  control={form.control}
                                  name={`headerRewrites.${index}.name`}
                                  render={({ field }) => (
                                    <FormItem>
                                      <FormControl>
                                        <Input {...field} />
                                      </FormControl>
                                      <FormMessage />
                                    </FormItem>
                                  )}
                                />
                              </TableCell>
                              <TableCell className="flex-grow">
                                <FormField
                                  control={form.control}
                                  name={`headerRewrites.${index}.value`}
                                  render={({ field }) => (
                                    <FormItem>
                                      <FormControl>
                                        <Input type={f.show ? "text" : "password"} {...field} />
                                      </FormControl>
                                      <FormMessage />
                                    </FormItem>
                                  )}
                                />
                              </TableCell>
                              <TableCell className="flex items-center flex-shrink gap-2 pr-0">
                                <Button
                                  type="button"
                                  variant="secondary"
                                  size="icon"
                                  onClick={() => {
                                    fields.update(index, { ...f, show: !f.show });
                                  }}
                                >
                                  {f.show ? (
                                    <Eye className="w-4 h-4" />
                                  ) : (
                                    <EyeOff className="w-4 h-4" />
                                  )}
                                </Button>
                                <Button
                                  type="button"
                                  variant="secondary"
                                  size="icon"
                                  onClick={() => {
                                    fields.remove(index);
                                  }}
                                >
                                  <X className="w-4 h-4" />
                                </Button>
                              </TableCell>
                            </TableRow>
                          ))}
                        </TableBody>
                      </Table>
                    ) : null}
                  </AccordionContent>
                </AccordionPrimitive.Item>
              </Accordion>

              <div className="w-full">
                <Button
                  className="w-full"
                  disabled={create.isLoading}
                  type="submit"
                  variant={create.isLoading || !form.formState.isValid ? "disabled" : "primary"}
                >
                  {create.isLoading ? <Loading /> : "Deploy"}
                </Button>
              </div>
            </form>
          </Form>
        </CardContent>
      </Card>
    </div>
  );
};
