"use client";
import {
  CommandDialog,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList,
} from "@/components/ui/command";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { trpc } from "@/lib/trpc/client";
import { zodResolver } from "@hookform/resolvers/zod";
import { BookOpen, LucideIcon, MessagesSquare } from "lucide-react";
import { useRouter } from "next/navigation";
import React, { useEffect, useState } from "react";
import { useForm } from "react-hook-form";
import { z } from "zod";
import { Button } from "../ui/button";
import { Form, FormControl, FormField, FormItem, FormLabel, FormMessage } from "../ui/form";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "../ui/select";
import { Textarea } from "../ui/textarea";
import { toast } from "../ui/toaster";
import { Loading } from "./loading";

export function CommandMenu() {
  const [open, setOpen] = React.useState(false);

  React.useEffect(() => {
    const down = (e: KeyboardEvent) => {
      if (e.key === "k" && (e.metaKey || e.ctrlKey)) {
        e.preventDefault();
        setOpen((open) => !open);
      }
    };

    document.addEventListener("keydown", down);
    return () => document.removeEventListener("keydown", down);
  }, []);
  return (
    <CommandDialog open={open} onOpenChange={setOpen}>
      <CommandInput placeholder="Type a command or search..." />
      <CommandList>
        <CommandEmpty>No results found.</CommandEmpty>

        <CommandGroup heading="Help">
          <DiscordCommand />
          <GenericLinkCommand
            close={() => setOpen(false)}
            href="/docs"
            label="Documentation"
            icon={BookOpen}
          />
          <Feedback />
        </CommandGroup>
      </CommandList>
    </CommandDialog>
  );
}

const DiscordCommand: React.FC = () => {
  const router = useRouter();
  return (
    <CommandItem onSelect={() => router.push("/discord")}>
      <svg
        className="w-4 h-4 mr-2"
        viewBox="0 -28.5 256 256"
        version="1.1"
        xmlns="http://www.w3.org/2000/svg"
        preserveAspectRatio="xMidYMid"
      >
        <g>
          <path
            d="M216.856339,16.5966031 C200.285002,8.84328665 182.566144,3.2084988 164.041564,0 C161.766523,4.11318106 159.108624,9.64549908 157.276099,14.0464379 C137.583995,11.0849896 118.072967,11.0849896 98.7430163,14.0464379 C96.9108417,9.64549908 94.1925838,4.11318106 91.8971895,0 C73.3526068,3.2084988 55.6133949,8.86399117 39.0420583,16.6376612 C5.61752293,67.146514 -3.4433191,116.400813 1.08711069,164.955721 C23.2560196,181.510915 44.7403634,191.567697 65.8621325,198.148576 C71.0772151,190.971126 75.7283628,183.341335 79.7352139,175.300261 C72.104019,172.400575 64.7949724,168.822202 57.8887866,164.667963 C59.7209612,163.310589 61.5131304,161.891452 63.2445898,160.431257 C105.36741,180.133187 151.134928,180.133187 192.754523,160.431257 C194.506336,161.891452 196.298154,163.310589 198.110326,164.667963 C191.183787,168.842556 183.854737,172.420929 176.223542,175.320965 C180.230393,183.341335 184.861538,190.991831 190.096624,198.16893 C211.238746,191.588051 232.743023,181.531619 254.911949,164.955721 C260.227747,108.668201 245.831087,59.8662432 216.856339,16.5966031 Z M85.4738752,135.09489 C72.8290281,135.09489 62.4592217,123.290155 62.4592217,108.914901 C62.4592217,94.5396472 72.607595,82.7145587 85.4738752,82.7145587 C98.3405064,82.7145587 108.709962,94.5189427 108.488529,108.914901 C108.508531,123.290155 98.3405064,135.09489 85.4738752,135.09489 Z M170.525237,135.09489 C157.88039,135.09489 147.510584,123.290155 147.510584,108.914901 C147.510584,94.5396472 157.658606,82.7145587 170.525237,82.7145587 C183.391518,82.7145587 193.761324,94.5189427 193.539891,108.914901 C193.539891,123.290155 183.391518,135.09489 170.525237,135.09489 Z"
            fill-rule="nonzero"
          />
        </g>
      </svg>
      <span>Go to Discord</span>
    </CommandItem>
  );
};

const GenericLinkCommand: React.FC<{
  href: string;
  label: string;
  icon: LucideIcon;
  close: () => void;
}> = (props) => {
  const router = useRouter();
  return (
    <CommandItem
      onSelect={() => {
        router.push(props.href);
        props.close();
      }}
    >
      <props.icon className="w-4 h-4 mr-2" />
      <span>{props.label}</span>
    </CommandItem>
  );
};

const Feedback: React.FC = () => {
  const [open, setOpen] = useState(false);
  /**
   * This was necessary cause otherwise the dialog would not close when you're clicking outside of it
   */
  const [selected, setSelected] = useState(false);
  useEffect(() => {
    if (selected) {
      setOpen(true);
    }
  }, [selected]);

  const schema = z.object({
    severity: z.enum(["p0", "p1", "p2", "p3"]),
    issueType: z.enum(["bug", "feature", "security", "payment", "question"]),
    message: z.string(),
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
      toast("Your issue has been created, we'll get back to you as soon as possible");
    },
    onError: (err) => {
      toast.error("Issue creation failed", {
        description: err.message,
      });
    },
  });

  return (
    <CommandItem
      onSelect={(_v) => {
        setSelected(true);
      }}
    >
      <MessagesSquare className="w-4 h-4 mr-2" />
      Feedback
      <Dialog open={open} onOpenChange={setOpen}>
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
                          <SelectItem value="p1">Hight</SelectItem>
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
    </CommandItem>
  );
};
