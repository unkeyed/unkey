"use client";
import { Loading } from "@/components/dashboard/loading";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Textarea } from "@/components/ui/textarea";
import { toast } from "@/components/ui/toaster";
import { trpc } from "@/lib/trpc/client";
import { zodResolver } from "@hookform/resolvers/zod";
import { Button } from "@unkey/ui";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@unkey/ui/src/components/select";
import { parseAsBoolean, useQueryState } from "nuqs";
import { useForm } from "react-hook-form";
import { z } from "zod";
import { Form, FormControl, FormField, FormItem, FormLabel, FormMessage } from "../ui/form";

export const useFeedback = () => {
  return useQueryState("feedback", {
    ...parseAsBoolean,
    clearOnDefault: true,
  });
};

export const Feedback: React.FC = () => {
  const [open, setOpen] = useFeedback();
  const schema = z.object({
    severity: z.enum(["p0", "p1", "p2", "p3"]),
    issueType: z.enum(["bug", "feature", "security", "payment", "question"]),
    message: z.string().min(20, "Feedback must contain at least 20 characters"),
  });

  const form = useForm<z.infer<typeof schema>>({
    resolver: zodResolver(schema),
    defaultValues: {
      severity: "p2",
      issueType: "bug",
      message: "",
    },
  });

  const create = trpc.plain.createIssue.useMutation({
    onSuccess: () => {
      setOpen(false);
      toast.success("Your issue has been created, we'll get back to you as soon as possible");
    },
    onError(err) {
      console.error(err);
      toast.error(err.message);
    },
  });

  return (
    <Dialog open={!!open} onOpenChange={setOpen}>
      <Form {...form}>
        <form>
          <DialogContent>
            <DialogHeader>
              <DialogTitle>Report an issue</DialogTitle>
              <DialogDescription>What went wrong or how can we improve?</DialogDescription>
            </DialogHeader>
            <div className="grid grid-cols-2 gap-4">
              <FormField
                control={form.control}
                name="issueType"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>Area</FormLabel>
                    <Select onValueChange={field.onChange} defaultValue={field.value}>
                      <FormControl>
                        <SelectTrigger>
                          <SelectValue placeholder="What area is this" />
                        </SelectTrigger>
                      </FormControl>
                      <SelectContent>
                        <SelectItem value="bug">Bug</SelectItem>
                        <SelectItem value="feature">Feature Request</SelectItem>
                        <SelectItem value="security">Security</SelectItem>
                        <SelectItem value="payment">Payments</SelectItem>
                        <SelectItem value="question">General Question</SelectItem>
                      </SelectContent>
                    </Select>

                    <FormMessage />
                  </FormItem>
                )}
              />
              <FormField
                control={form.control}
                name="severity"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>Severity</FormLabel>
                    <Select onValueChange={field.onChange} defaultValue={field.value}>
                      <FormControl>
                        <SelectTrigger>
                          <SelectValue placeholder="Select a severity" />
                        </SelectTrigger>
                      </FormControl>
                      <SelectContent>
                        <SelectItem value="p0">Urgent</SelectItem>
                        <SelectItem value="p1">High</SelectItem>
                        <SelectItem value="p2">Normal</SelectItem>
                        <SelectItem value="p3">Low</SelectItem>
                      </SelectContent>
                    </Select>

                    <FormMessage />
                  </FormItem>
                )}
              />
            </div>
            <FormField
              control={form.control}
              name="message"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>What can we do for you?</FormLabel>
                  <FormControl>
                    <Textarea
                      {...field}
                      placeholder="Please include all information relevant to your issue."
                    />
                  </FormControl>

                  <FormMessage />
                </FormItem>
              )}
            />

            <DialogFooter>
              <Button variant="ghost" onClick={() => setOpen(false)}>
                Cancel
              </Button>
              <Button
                type="submit"
                disabled={create.isLoading}
                onClick={form.handleSubmit((data) => {
                  create.mutate(data);
                })}
              >
                {create.isLoading ? <Loading /> : "Send"}
              </Button>
            </DialogFooter>
          </DialogContent>
        </form>
      </Form>
    </Dialog>
  );
};
