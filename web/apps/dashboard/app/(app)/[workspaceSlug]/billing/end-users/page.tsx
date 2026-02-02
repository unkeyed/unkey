"use client";

import { PageLoading } from "@/components/dashboard/page-loading";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { trpc } from "@/lib/trpc/client";
import { zodResolver } from "@hookform/resolvers/zod";
import { Eye, PenWriting3, Plus, Trash } from "@unkey/icons";
import {
  Button,
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  Empty,
  FormInput,
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
  SettingCard,
  toast,
} from "@unkey/ui";
import { useState } from "react";
import { useForm } from "react-hook-form";
import { z } from "zod";
import { BillingNavbar } from "../billing-navbar";
import { EndUserExternalIdField } from "./components/external-id-field";

interface EndUserItem {
  id: string;
  externalId: string;
  pricingModelId: string;
  stripeCustomerId: string;
  email: string | null;
  name: string | null;
  createdAtM: number;
  pricingModel: {
    id: string;
    name: string;
    currency: string;
  } | null;
}

interface InvoiceItem {
  id: string;
  totalAmount: number;
  currency: string;
  status: string;
}

interface PricingModelOption {
  id: string;
  name: string;
  currency: string;
}

const endUserSchema = z.object({
  externalId: z.string().min(1, "External ID is required").max(255),
  pricingModelId: z.string().min(1, "Pricing model is required"),
  email: z.string().email().optional().or(z.literal("")),
  name: z.string().max(255).optional(),
});

type EndUserFormData = z.infer<typeof endUserSchema>;

export default function EndUsersPage() {
  const workspace = useWorkspaceNavigation();
  const [isCreateOpen, setIsCreateOpen] = useState(false);
  const [editingUser, setEditingUser] = useState<string | null>(null);
  const [viewingUser, setViewingUser] = useState<string | null>(null);

  const utils = trpc.useUtils();

  const { data: endUsers, isLoading } = trpc.customerBilling.endUsers.list.useQuery();
  const { data: pricingModels } = trpc.customerBilling.pricingModels.list.useQuery();
  const { data: connectedAccount } = trpc.customerBilling.connect.getAccount.useQuery();
  const { data: viewedUser, isLoading: isLoadingUser } = trpc.customerBilling.endUsers.get.useQuery(
    { id: viewingUser ?? "" },
    { enabled: !!viewingUser },
  );

  const createMutation = trpc.customerBilling.endUsers.create.useMutation({
    onSuccess: () => {
      toast.success("End user created");
      utils.customerBilling.endUsers.list.invalidate();
      setIsCreateOpen(false);
      setSelectedIdentityId(null);
      createForm.reset();
    },
    onError: (error: { message: string }) => {
      toast.error("Failed to create end user", {
        description: error.message,
      });
    },
  });

  const updateMutation = trpc.customerBilling.endUsers.update.useMutation({
    onSuccess: () => {
      toast.success("End user updated");
      utils.customerBilling.endUsers.list.invalidate();
      setEditingUser(null);
      editForm.reset();
    },
    onError: (error: { message: string }) => {
      toast.error("Failed to update end user", {
        description: error.message,
      });
    },
  });

  const deleteMutation = trpc.customerBilling.endUsers.delete.useMutation({
    onSuccess: () => {
      toast.success("End user deleted");
      utils.customerBilling.endUsers.list.invalidate();
    },
    onError: (error: { message: string }) => {
      toast.error("Failed to delete end user", {
        description: error.message,
      });
    },
  });

  const [selectedIdentityId, setSelectedIdentityId] = useState<string | null>(null);

  const createForm = useForm<EndUserFormData>({
    resolver: zodResolver(endUserSchema),
    defaultValues: {
      externalId: "",
      pricingModelId: "",
      email: "",
      name: "",
    },
  });

  const editForm = useForm<EndUserFormData>({
    resolver: zodResolver(endUserSchema),
  });

  const handleCreate = (data: EndUserFormData) => {
    createMutation.mutate({
      externalId: data.externalId.trim(),
      pricingModelId: data.pricingModelId,
      email: data.email || undefined,
      name: data.name || undefined,
    });
  };

  const handleUpdate = (data: EndUserFormData) => {
    if (!editingUser) {
      return;
    }
    updateMutation.mutate({
      id: editingUser,
      pricingModelId: data.pricingModelId,
      email: data.email || undefined,
      name: data.name || undefined,
    });
  };

  const handleDelete = (id: string) => {
    if (confirm("Are you sure you want to delete this end user?")) {
      deleteMutation.mutate({ id });
    }
  };

  const openEditDialog = (user: NonNullable<typeof endUsers>[number]) => {
    setEditingUser(user.id);
    editForm.reset({
      externalId: user.externalId,
      pricingModelId: user.pricingModelId,
      email: user.email ?? "",
      name: user.name ?? "",
    });
  };

  // Check if billing beta is enabled
  if (!workspace.betaFeatures.billing) {
    return (
      <div>
        <BillingNavbar activePage={{ href: "end-users", text: "End Users" }} />
        <div className="p-4">
          <Empty>
            <Empty.Icon />
            <Empty.Title>Customer Billing Not Enabled</Empty.Title>
            <Empty.Description>
              Customer billing is currently in beta. Contact support to enable this feature.
            </Empty.Description>
          </Empty>
        </div>
      </div>
    );
  }

  if (isLoading) {
    return <PageLoading message="Loading end users..." />;
  }

  const isConnected = !!connectedAccount;
  const hasPricingModels = pricingModels && pricingModels.length > 0;

  return (
    <div>
      <BillingNavbar activePage={{ href: "end-users", text: "End Users" }} />
      <div className="py-3 w-full flex items-center justify-center">
        <div className="w-[900px] flex flex-col justify-center items-center gap-5 mx-6 mt-4">
          <div className="w-full flex justify-between items-center">
            <h2 className="text-lg font-medium">End Users</h2>
            <Button
              variant="primary"
              onClick={() => setIsCreateOpen(true)}
              disabled={!isConnected || !hasPricingModels}
            >
              <Plus className="w-4 h-4 mr-2" />
              Add End User
            </Button>
          </div>

          {endUsers && endUsers.length > 0 ? (
            <div className="w-full flex flex-col">
              {(endUsers as EndUserItem[]).map((user: EndUserItem, index: number) => (
                <SettingCard
                  key={user.id}
                  title={user.name || user.externalId}
                  description={
                    <div className="flex flex-col gap-1">
                      <span>External ID: {user.externalId}</span>
                      {user.email && <span>Email: {user.email}</span>}
                      <span>Pricing: {user.pricingModel?.name ?? "Unknown"}</span>
                    </div>
                  }
                  border={index === 0 ? "top" : index === endUsers.length - 1 ? "bottom" : "none"}
                >
                  <div className="flex items-center gap-2 w-full justify-end">
                    <Button variant="outline" size="sm" onClick={() => setViewingUser(user.id)}>
                      <Eye className="w-4 h-4" />
                    </Button>
                    <Button variant="outline" size="sm" onClick={() => openEditDialog(user)}>
                      <PenWriting3 className="w-4 h-4" />
                    </Button>
                    <Button
                      variant="destructive"
                      size="sm"
                      onClick={() => handleDelete(user.id)}
                      loading={deleteMutation.isLoading}
                    >
                      <Trash className="w-4 h-4" />
                    </Button>
                  </div>
                </SettingCard>
              ))}
            </div>
          ) : (
            <Empty className="w-full">
              <Empty.Icon />
              <Empty.Title>No End Users</Empty.Title>
              <Empty.Description>
                Add end users to start tracking their usage and generating invoices.
              </Empty.Description>
            </Empty>
          )}
        </div>
      </div>

      {/* Create Dialog */}
      <Dialog
        open={isCreateOpen}
        onOpenChange={(open) => {
          if (!open) {
            setSelectedIdentityId(null);
            createForm.reset();
          }
          setIsCreateOpen(open);
        }}
      >
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Add End User</DialogTitle>
          </DialogHeader>
          <form onSubmit={createForm.handleSubmit(handleCreate)} className="space-y-4">
            <EndUserExternalIdField
              value={selectedIdentityId}
              onChange={(identityId, externalId) => {
                setSelectedIdentityId(identityId);
                if (externalId) {
                  createForm.setValue("externalId", externalId, { shouldValidate: true });
                }
              }}
              onSelectIdentity={(identityId, externalId) => {
                setSelectedIdentityId(identityId);
                if (externalId) {
                  createForm.setValue("externalId", externalId, { shouldValidate: true });
                }
              }}
              error={createForm.formState.errors.externalId?.message}
            />

            <div className="space-y-1.5">
              <span className="text-sm font-medium">Pricing Model</span>
              <Select
                value={createForm.watch("pricingModelId")}
                onValueChange={(value) => createForm.setValue("pricingModelId", value)}
              >
                <SelectTrigger>
                  <SelectValue placeholder="Select pricing model" />
                </SelectTrigger>
                <SelectContent>
                  {(pricingModels as PricingModelOption[] | undefined)?.map(
                    (model: PricingModelOption) => (
                      <SelectItem key={model.id} value={model.id}>
                        {model.name} ({model.currency})
                      </SelectItem>
                    ),
                  )}
                </SelectContent>
              </Select>
              {createForm.formState.errors.pricingModelId && (
                <p className="text-xs text-error-11">
                  {createForm.formState.errors.pricingModelId.message}
                </p>
              )}
            </div>

            <FormInput
              label="Email (optional)"
              type="email"
              placeholder="user@example.com"
              {...createForm.register("email")}
              error={createForm.formState.errors.email?.message}
            />

            <FormInput
              label="Name (optional)"
              placeholder="John Doe"
              {...createForm.register("name")}
              error={createForm.formState.errors.name?.message}
            />

            <DialogFooter>
              <Button type="button" variant="outline" onClick={() => setIsCreateOpen(false)}>
                Cancel
              </Button>
              <Button type="submit" variant="primary" loading={createMutation.isLoading}>
                Create
              </Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>

      {/* Edit Dialog */}
      <Dialog open={!!editingUser} onOpenChange={(open) => !open && setEditingUser(null)}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Edit End User</DialogTitle>
          </DialogHeader>
          <form onSubmit={editForm.handleSubmit(handleUpdate)} className="space-y-4">
            <FormInput label="External ID" disabled {...editForm.register("externalId")} />

            <div className="space-y-1.5">
              <span className="text-sm font-medium">Pricing Model</span>
              <Select
                value={editForm.watch("pricingModelId")}
                onValueChange={(value) => editForm.setValue("pricingModelId", value)}
              >
                <SelectTrigger>
                  <SelectValue placeholder="Select pricing model" />
                </SelectTrigger>
                <SelectContent>
                  {(pricingModels as PricingModelOption[] | undefined)?.map(
                    (model: PricingModelOption) => (
                      <SelectItem key={model.id} value={model.id}>
                        {model.name} ({model.currency})
                      </SelectItem>
                    ),
                  )}
                </SelectContent>
              </Select>
              <p className="text-xs text-gray-11">
                Note: Pricing model changes apply to future billing periods only.
              </p>
            </div>

            <FormInput
              label="Email (optional)"
              type="email"
              placeholder="user@example.com"
              {...editForm.register("email")}
              error={editForm.formState.errors.email?.message}
            />

            <FormInput
              label="Name (optional)"
              placeholder="John Doe"
              {...editForm.register("name")}
              error={editForm.formState.errors.name?.message}
            />

            <DialogFooter>
              <Button type="button" variant="outline" onClick={() => setEditingUser(null)}>
                Cancel
              </Button>
              <Button type="submit" variant="primary" loading={updateMutation.isLoading}>
                Update
              </Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>

      {/* View Dialog */}
      <Dialog open={!!viewingUser} onOpenChange={(open) => !open && setViewingUser(null)}>
        <DialogContent className="max-w-2xl">
          <DialogHeader>
            <DialogTitle>End User Details</DialogTitle>
          </DialogHeader>
          {isLoadingUser ? (
            <div className="py-8 text-center text-gray-11">Loading...</div>
          ) : viewedUser ? (
            <div className="space-y-4">
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <span className="text-sm text-gray-11">External ID</span>
                  <p className="font-mono">{viewedUser.externalId}</p>
                </div>
                <div>
                  <span className="text-sm text-gray-11">Stripe Customer ID</span>
                  <p className="font-mono text-sm">{viewedUser.stripeCustomerId}</p>
                </div>
                <div>
                  <span className="text-sm text-gray-11">Name</span>
                  <p>{viewedUser.name || "-"}</p>
                </div>
                <div>
                  <span className="text-sm text-gray-11">Email</span>
                  <p>{viewedUser.email || "-"}</p>
                </div>
                <div>
                  <span className="text-sm text-gray-11">Pricing Model</span>
                  <p>{viewedUser.pricingModel?.name ?? "Unknown"}</p>
                </div>
                <div>
                  <span className="text-sm text-gray-11">Created</span>
                  <p>{new Date(viewedUser.createdAtM).toLocaleDateString()}</p>
                </div>
              </div>

              {viewedUser.invoices && viewedUser.invoices.length > 0 && (
                <div>
                  <h4 className="text-sm font-medium mb-2">Recent Invoices</h4>
                  <div className="border rounded-lg divide-y">
                    {(viewedUser.invoices as InvoiceItem[]).map((invoice: InvoiceItem) => (
                      <div key={invoice.id} className="p-3 flex justify-between items-center">
                        <div>
                          <span className="font-mono text-sm">{invoice.id}</span>
                          <span className="ml-2 text-sm text-gray-11">
                            ${(invoice.totalAmount / 100).toFixed(2)} {invoice.currency}
                          </span>
                        </div>
                        <span
                          className={`text-sm px-2 py-1 rounded ${
                            invoice.status === "paid"
                              ? "bg-success-3 text-success-11"
                              : invoice.status === "open"
                                ? "bg-warning-3 text-warning-11"
                                : "bg-gray-3 text-gray-11"
                          }`}
                        >
                          {invoice.status}
                        </span>
                      </div>
                    ))}
                  </div>
                </div>
              )}
            </div>
          ) : null}
          <DialogFooter>
            <Button variant="outline" onClick={() => setViewingUser(null)}>
              Close
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
