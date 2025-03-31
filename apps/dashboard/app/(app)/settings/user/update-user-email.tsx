"use client";

/**
 * TODO: Remove or re-work this
 * WorkOS doesn't allow users to update their email
 */

import { Loading } from "@/components/dashboard/loading";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent, CardFooter, CardHeader, CardTitle } from "@/components/ui/card";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuShortcut,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Form, FormControl, FormField, FormItem, FormMessage } from "@/components/ui/form";
import { Input } from "@/components/ui/input";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { toast } from "@/components/ui/toaster";
import type { ClerkError } from "@/lib/clerk";
import { useClerk, useUser } from "@clerk/nextjs";
import { zodResolver } from "@hookform/resolvers/zod";
import { Empty } from "@unkey/ui";
import { Button } from "@unkey/ui";
import { ChevronsUp, MoreHorizontal, ShieldCheck, X } from "lucide-react";
import React, { useEffect, useState } from "react";
import { useForm } from "react-hook-form";
import { z } from "zod";
const formSchema = z.object({
  email: z.string().email(),
});

const verificationSchema = z.object({
  code: z.string().min(6).max(6),
});

export const UpdateUserEmail: React.FC = () => {
  const { user } = useUser();
  const [sendingVerification, setSendingVerification] = useState(false);
  const [resetPointerEvents, setResetPointerEvents] = useState(false);
  const [promotingEmail, setPromotingEmail] = useState(false);
  const [openRemoveModal, setOpenRemoveModal] = React.useState(false);
  const [verifyEmail, setVerifyEmail] = React.useState<string | null>(null);
  const emailForm = useForm<z.infer<typeof formSchema>>({
    resolver: zodResolver(formSchema),
    mode: "all",
    defaultValues: {
      email: user?.primaryEmailAddress?.emailAddress ?? "",
    },
  });

  // https://github.com/radix-ui/primitives/issues/1241#issuecomment-1888232392
  useEffect(() => {
    if (resetPointerEvents) {
      setTimeout(() => {
        document.body.style.pointerEvents = "";
      });
    }
  }, [resetPointerEvents]);

  if (!user) {
    return (
      <Empty>
        <Loading />
      </Empty>
    );
  }
  const isDisabled = emailForm.formState.isLoading || !emailForm.formState.isValid;
  const verifiedEmails = user.emailAddresses.filter(
    (email) => email.verification.status === "verified",
  ).length;

  return (
    <>
      <Card>
        <CardHeader>
          <CardTitle>Email Addresses</CardTitle>
        </CardHeader>
        <CardContent>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Email</TableHead>
                <TableHead>Primary</TableHead>
                <TableHead>Status</TableHead>
                <TableHead className="text-right">Settings</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {user.emailAddresses?.map(
                ({ id, emailAddress, verification, destroy, prepareVerification }) => (
                  <TableRow key={id}>
                    <TableCell>{emailAddress}</TableCell>
                    <TableCell>
                      {user.primaryEmailAddress?.id === id ? (
                        <Badge size="sm" variant="secondary">
                          Primary
                        </Badge>
                      ) : null}
                    </TableCell>
                    <TableCell>
                      <Badge
                        size="sm"
                        variant={verification.status === "verified" ? "secondary" : "alert"}
                        className="capitalize"
                      >
                        {verification.status}
                      </Badge>
                    </TableCell>

                    <TableCell align="right">
                      <DropdownMenu>
                        <DropdownMenuTrigger>
                          <Button variant="ghost">
                            <MoreHorizontal className="w-4 h-4" />
                          </Button>
                        </DropdownMenuTrigger>
                        <DropdownMenuContent className="w-56">
                          <DropdownMenuGroup>
                            <Dialog
                              open={openRemoveModal}
                              onOpenChange={(o) => setOpenRemoveModal(o)}
                            >
                              <DialogTrigger asChild>
                                <DropdownMenuItem
                                  disabled={
                                    verifiedEmails <= 1 && verification.status === "verified"
                                  }
                                  onClick={(e) => {
                                    e.preventDefault();
                                    setOpenRemoveModal(true);
                                  }}
                                >
                                  Remove
                                  <DropdownMenuShortcut>
                                    <X className="w-4 h-4" />
                                  </DropdownMenuShortcut>
                                </DropdownMenuItem>
                              </DialogTrigger>
                              <DialogContent className="sm:max-w-[425px] border-alert">
                                <DialogHeader>
                                  <DialogTitle>Remove Email</DialogTitle>
                                  <DialogDescription>
                                    Are you sure you want to remove {emailAddress}?
                                  </DialogDescription>
                                </DialogHeader>

                                <DialogFooter>
                                  <Button
                                    type="submit"
                                    variant="destructive"
                                    onClick={() => {
                                      destroy()
                                        .then(() => {
                                          toast.success("Email removed");
                                          user.reload();
                                          setResetPointerEvents(true);
                                        })
                                        .catch((e) => {
                                          toast.error((e as Error).message);
                                        });
                                    }}
                                  >
                                    Confirm
                                  </Button>
                                </DialogFooter>
                              </DialogContent>
                            </Dialog>
                            <DropdownMenuItem
                              disabled={user.primaryEmailAddress?.id === id}
                              onClick={async () => {
                                try {
                                  setPromotingEmail(true);
                                  await user.update({ primaryEmailAddressId: id });
                                  user.reload();
                                } catch (e) {
                                  toast.error((e as Error).message);
                                } finally {
                                  setPromotingEmail(false);
                                }
                              }}
                            >
                              Make Primary
                              <DropdownMenuShortcut>
                                {" "}
                                {promotingEmail ? (
                                  <Loading className="w-4 h-4" />
                                ) : (
                                  <ChevronsUp className="w-4 h-4" />
                                )}
                              </DropdownMenuShortcut>
                            </DropdownMenuItem>
                            <DropdownMenuItem
                              disabled={verification.status === "verified"}
                              onClick={async () => {
                                try {
                                  setSendingVerification(true);
                                  await prepareVerification({
                                    strategy: "email_code",
                                  });
                                  setVerifyEmail(emailAddress);
                                } catch (e) {
                                  toast.error((e as Error).message);
                                } finally {
                                  setSendingVerification(false);
                                }
                              }}
                            >
                              Verify
                              <DropdownMenuShortcut>
                                {sendingVerification ? (
                                  <Loading className="w-4 h-4" />
                                ) : (
                                  <ShieldCheck className="w-4 h-4" />
                                )}
                              </DropdownMenuShortcut>
                            </DropdownMenuItem>
                          </DropdownMenuGroup>
                        </DropdownMenuContent>
                      </DropdownMenu>
                    </TableCell>
                  </TableRow>
                ),
              )}
            </TableBody>
          </Table>
        </CardContent>
        <CardFooter className="justify-end">
          <Form {...emailForm}>
            <form
              onSubmit={emailForm.handleSubmit(async ({ email }) => {
                try {
                  setSendingVerification(true);
                  const emailResponse = await user.createEmailAddress({ email });

                  await emailResponse.prepareVerification({
                    strategy: "email_code",
                  });

                  setVerifyEmail(email);
                } catch (e) {
                  toast.error(
                    (e as ClerkError)?.errors.at(0)?.longMessage ?? "Error creating email address",
                  );
                } finally {
                  setSendingVerification(false);
                }
              })}
              className="flex items-start justify-start gap-4"
            >
              <FormField
                control={emailForm.control}
                name="email"
                render={({ field }) => (
                  <FormItem>
                    <FormControl>
                      <Input {...field} className="max-w-md min-w-md" placeholder="Add new email" />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />

              <Button type="submit" variant="primary" disabled={isDisabled || sendingVerification}>
                {emailForm.formState.isLoading || sendingVerification ? <Loading /> : "Save"}
              </Button>
            </form>
          </Form>
        </CardFooter>
      </Card>

      <Dialog
        open={!!verifyEmail}
        onOpenChange={(o) => {
          setVerifyEmail(o ? verifyEmail : null);
        }}
      >
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Verify your email</DialogTitle>
            <DialogDescription>We have sent a code to your email</DialogDescription>
          </DialogHeader>

          <VerificationForm
            email={emailForm.getValues().email}
            onSuccess={() => {
              setVerifyEmail(null);
              emailForm.reset();
              user.reload();
            }}
          />
        </DialogContent>
      </Dialog>
    </>
  );
};

type VerificationFormProps = {
  email: string;
  onSuccess: () => void;
};

const VerificationForm: React.FC<VerificationFormProps> = ({ email, onSuccess }) => {
  const { user } = useClerk();
  const verificationForm = useForm<z.infer<typeof verificationSchema>>({
    resolver: zodResolver(verificationSchema),
    mode: "onSubmit",
  });
  if (!user) {
    return null;
  }
  return (
    <Form {...verificationForm}>
      <form
        onSubmit={verificationForm.handleSubmit(async ({ code }) => {
          try {
            const emailResource = user.emailAddresses.find((e) => e.emailAddress === email);
            if (!emailResource) {
              throw new Error("Invalid email");
            }
            const verify = await emailResource.attemptVerification({ code });
            if (verify.verification.status !== "verified") {
              throw new Error("Invalid verification code");
            }

            onSuccess();
          } catch (e) {
            toast.error((e as Error).message);
          }
        })}
      >
        <div className="flex items-start justify-between w-full gap-4">
          <div className="w-full">
            <FormField
              control={verificationForm.control}
              name="code"
              render={({ field }) => (
                <FormItem>
                  <FormControl>
                    <Input
                      {...field}
                      className="w-full grow"
                      placeholder="Enter the 6 digit code here"
                      autoComplete="off"
                    />
                  </FormControl>

                  <FormMessage />
                </FormItem>
              )}
            />
          </div>
          <Button type="submit" variant="primary" disabled={verificationForm.formState.isLoading}>
            {verificationForm.formState.isLoading ? <Loading /> : "Verify"}
          </Button>
        </div>
      </form>
    </Form>
  );
};
