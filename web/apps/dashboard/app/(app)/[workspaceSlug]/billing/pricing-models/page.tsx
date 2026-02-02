"use client";

import { PageLoading } from "@/components/dashboard/page-loading";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { trpc } from "@/lib/trpc/client";
import { zodResolver } from "@hookform/resolvers/zod";
import { CircleInfo, PenWriting3, Plus, Trash } from "@unkey/icons";
import {
  Alert,
  AlertDescription,
  AlertTitle,
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
  Switch,
  toast,
} from "@unkey/ui";
import { useState } from "react";
import { useForm } from "react-hook-form";
import { z } from "zod";
import { BillingNavbar } from "../billing-navbar";

const CURRENCIES = [
  { value: "USD", label: "USD - US Dollar" },
  { value: "EUR", label: "EUR - Euro" },
  { value: "GBP", label: "GBP - British Pound" },
  { value: "CAD", label: "CAD - Canadian Dollar" },
  { value: "AUD", label: "AUD - Australian Dollar" },
];

const pricingModelSchema = z.object({
  name: z.string().min(1, "Name is required").max(255),
  currency: z.string().length(3, "Currency must be 3 characters"),
  enableKeyAccess: z.boolean(),
  keyAccessUnitPrice: z.number().min(0, "Price must be non-negative").optional(),
  enableVerification: z.boolean(),
  verificationUnitPrice: z.number().min(0, "Price must be non-negative").optional(),
  enableCredit: z.boolean(),
  creditUnitPrice: z.number().min(0, "Price must be non-negative").optional(),
});

type PricingModelFormData = z.infer<typeof pricingModelSchema>;

interface PricingModelItem {
  id: string;
  name: string;
  currency: string;
  verificationUnitPrice: number;
  keyAccessUnitPrice: number;
  creditUnitPrice: number;
  version: number;
}

export default function PricingModelsPage() {
  const workspace = useWorkspaceNavigation();
  const [isCreateOpen, setIsCreateOpen] = useState(false);
  const [editingModel, setEditingModel] = useState<string | null>(null);

  const utils = trpc.useUtils();

  const { data: pricingModels, isLoading } = trpc.customerBilling.pricingModels.list.useQuery();
  const { data: workspaceCurrency } =
    trpc.customerBilling.pricingModels.getWorkspaceCurrency.useQuery();
  const { data: connectedAccount } = trpc.customerBilling.connect.getAccount.useQuery();

  const createMutation = trpc.customerBilling.pricingModels.create.useMutation({
    onSuccess: () => {
      toast.success("Pricing model created");
      utils.customerBilling.pricingModels.list.invalidate();
      utils.customerBilling.pricingModels.getWorkspaceCurrency.invalidate();
      setIsCreateOpen(false);
      createForm.reset();
    },
    onError: (error: { message: string }) => {
      toast.error("Failed to create pricing model", {
        description: error.message,
      });
    },
  });

  const updateMutation = trpc.customerBilling.pricingModels.update.useMutation({
    onSuccess: () => {
      toast.success("Pricing model updated");
      utils.customerBilling.pricingModels.list.invalidate();
      setEditingModel(null);
      editForm.reset();
    },
    onError: (error: { message: string }) => {
      toast.error("Failed to update pricing model", {
        description: error.message,
      });
    },
  });

  const deleteMutation = trpc.customerBilling.pricingModels.delete.useMutation({
    onSuccess: () => {
      toast.success("Pricing model deleted");
      utils.customerBilling.pricingModels.list.invalidate();
    },
    onError: (error: { message: string }) => {
      toast.error("Failed to delete pricing model", {
        description: error.message,
      });
    },
  });

  const createForm = useForm<PricingModelFormData>({
    resolver: zodResolver(pricingModelSchema),
    defaultValues: {
      name: "",
      currency: workspaceCurrency ?? "USD",
      enableKeyAccess: false,
      keyAccessUnitPrice: 0,
      enableVerification: true,
      verificationUnitPrice: 0,
      enableCredit: false,
      creditUnitPrice: 0,
    },
  });

  const editForm = useForm<PricingModelFormData>({
    resolver: zodResolver(pricingModelSchema),
  });

  const handleCreate = (data: PricingModelFormData) => {
    createMutation.mutate({
      name: data.name,
      currency: data.currency,
      verificationUnitPrice: data.enableVerification ? Math.round((data.verificationUnitPrice ?? 0) * 100) : 0,
      keyAccessUnitPrice: data.enableKeyAccess ? Math.round((data.keyAccessUnitPrice ?? 0) * 100) : 0,
      creditUnitPrice: data.enableCredit ? Math.round((data.creditUnitPrice ?? 0) * 100) : 0,
    });
  };

  const handleUpdate = (data: PricingModelFormData) => {
    if (!editingModel) {
      return;
    }
    updateMutation.mutate({
      id: editingModel,
      name: data.name,
      verificationUnitPrice: data.enableVerification ? Math.round((data.verificationUnitPrice ?? 0) * 100) : 0,
      keyAccessUnitPrice: data.enableKeyAccess ? Math.round((data.keyAccessUnitPrice ?? 0) * 100) : 0,
      creditUnitPrice: data.enableCredit ? Math.round((data.creditUnitPrice ?? 0) * 100) : 0,
    });
  };

  const handleDelete = (id: string) => {
    if (confirm("Are you sure you want to delete this pricing model?")) {
      deleteMutation.mutate({ id });
    }
  };

  const openEditDialog = (model: PricingModelItem) => {
    setEditingModel(model.id);
    editForm.reset({
      name: model.name,
      currency: model.currency,
      enableKeyAccess: model.keyAccessUnitPrice > 0,
      keyAccessUnitPrice: model.keyAccessUnitPrice / 100,
      enableVerification: model.verificationUnitPrice > 0,
      verificationUnitPrice: model.verificationUnitPrice / 100,
      enableCredit: model.creditUnitPrice > 0,
      creditUnitPrice: model.creditUnitPrice / 100,
    });
  };

  if (!workspace.betaFeatures.billing) {
    return (
      <div>
        <BillingNavbar activePage={{ href: "pricing-models", text: "Pricing Models" }} />
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
    return <PageLoading message="Loading pricing models..." />;
  }

  const isConnected = !!connectedAccount;

  return (
    <div>
      <BillingNavbar activePage={{ href: "pricing-models", text: "Pricing Models" }} />
      <div className="py-3 w-full flex items-center justify-center">
        <div className="w-[900px] flex flex-col justify-center items-center gap-5 mx-6 mt-4">
          {workspaceCurrency && (
            <Alert variant="default" className="w-full">
              <CircleInfo className="h-4 w-4" />
              <AlertTitle>Workspace Currency: {workspaceCurrency}</AlertTitle>
              <AlertDescription>
                All pricing models in this workspace must use {workspaceCurrency}.
              </AlertDescription>
            </Alert>
          )}

          <div className="w-full flex justify-between items-center">
            <h2 className="text-lg font-medium">Pricing Models</h2>
            <Button variant="primary" onClick={() => setIsCreateOpen(true)} disabled={!isConnected}>
              <Plus className="w-4 h-4 mr-2" />
              Create Pricing Model
            </Button>
          </div>

          {pricingModels && pricingModels.length > 0 ? (
            <div className="w-full flex flex-col">
              {pricingModels.map((model: PricingModelItem, index: number) => (
                <SettingCard
                  key={model.id}
                  title={model.name}
                  description={`${model.currency} • Version ${model.version}`}
                  border={
                    index === 0 ? "top" : index === pricingModels.length - 1 ? "bottom" : "none"
                  }
                >
                  <div className="flex items-center gap-4 w-full justify-end">
                    <div className="text-sm text-gray-11">
                      {model.keyAccessUnitPrice > 0 && (
                        <div>Key Access: ${(model.keyAccessUnitPrice / 100).toFixed(4)}/key</div>
                      )}
                      {model.verificationUnitPrice > 0 && (
                        <div>Verification: ${(model.verificationUnitPrice / 100).toFixed(4)}/unit</div>
                      )}
                      {model.creditUnitPrice > 0 && (
                        <div>Credit: ${(model.creditUnitPrice / 100).toFixed(4)}/credit</div>
                      )}
                    </div>
                    <div className="flex gap-2">
                      <Button variant="outline" size="sm" onClick={() => openEditDialog(model)}>
                        <PenWriting3 className="w-4 h-4" />
                      </Button>
                      <Button
                        variant="destructive"
                        size="sm"
                        onClick={() => handleDelete(model.id)}
                        loading={deleteMutation.isLoading}
                      >
                        <Trash className="w-4 h-4" />
                      </Button>
                    </div>
                  </div>
                </SettingCard>
              ))}
            </div>
          ) : (
            <Empty className="w-full">
              <Empty.Icon />
              <Empty.Title>No Pricing Models</Empty.Title>
              <Empty.Description>
                Create a pricing model to define how you'll charge your end users.
              </Empty.Description>
            </Empty>
          )}
        </div>
      </div>

      <Dialog open={isCreateOpen} onOpenChange={setIsCreateOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Create Pricing Model</DialogTitle>
          </DialogHeader>
          <form onSubmit={createForm.handleSubmit(handleCreate)} className="space-y-4">
            <FormInput
              label="Name"
              placeholder="e.g., Standard Plan"
              {...createForm.register("name")}
              error={createForm.formState.errors.name?.message}
            />

            <div className="space-y-1.5">
              <span className="text-sm font-medium">Currency</span>
              <Select
                value={createForm.watch("currency")}
                onValueChange={(value) => createForm.setValue("currency", value)}
                disabled={!!workspaceCurrency}
              >
                <SelectTrigger>
                  <SelectValue placeholder="Select currency" />
                </SelectTrigger>
                <SelectContent>
                  {CURRENCIES.map((currency) => (
                    <SelectItem
                      key={currency.value}
                      value={currency.value}
                      disabled={workspaceCurrency ? currency.value !== workspaceCurrency : false}
                    >
                      {currency.label}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
              {workspaceCurrency && (
                <p className="text-xs text-gray-11">
                  Currency is locked to {workspaceCurrency} for this workspace.
                </p>
              )}
            </div>

            <div className="border rounded-lg p-4 space-y-4">
              <div className="flex items-center justify-between">
                <div>
                  <span className="font-medium">Key Access</span>
                  <p className="text-xs text-gray-11">Charge per unique key with access</p>
                </div>
                <Switch
                  checked={createForm.watch("enableKeyAccess")}
                  onCheckedChange={(checked) => createForm.setValue("enableKeyAccess", checked)}
                />
              </div>
              {createForm.watch("enableKeyAccess") && (
                <FormInput
                  label="Key Access Price (per key)"
                  type="number"
                  step="0.0001"
                  min="0"
                  placeholder="0.0001"
                  {...createForm.register("keyAccessUnitPrice", { valueAsNumber: true })}
                  error={createForm.formState.errors.keyAccessUnitPrice?.message}
                />
              )}
            </div>

            <div className="border rounded-lg p-4 space-y-4">
              <div className="flex items-center justify-between">
                <div>
                  <span className="font-medium">Verifications</span>
                  <p className="text-xs text-gray-11">Charge per API verification</p>
                </div>
                <Switch
                  checked={createForm.watch("enableVerification")}
                  onCheckedChange={(checked) => createForm.setValue("enableVerification", checked)}
                />
              </div>
              {createForm.watch("enableVerification") && (
                <FormInput
                  label="Verification Price (per unit)"
                  type="number"
                  step="0.0001"
                  min="0"
                  placeholder="0.0001"
                  {...createForm.register("verificationUnitPrice", { valueAsNumber: true })}
                  error={createForm.formState.errors.verificationUnitPrice?.message}
                />
              )}
            </div>

            <div className="border rounded-lg p-4 space-y-4">
              <div className="flex items-center justify-between">
                <div>
                  <span className="font-medium">Credits</span>
                  <p className="text-xs text-gray-11">Charge per credit</p>
                </div>
                <Switch
                  checked={createForm.watch("enableCredit")}
                  onCheckedChange={(checked) => createForm.setValue("enableCredit", checked)}
                />
              </div>
              {createForm.watch("enableCredit") && (
                <FormInput
                  label="Credit Price (per credit)"
                  type="number"
                  step="0.0001"
                  min="0"
                  placeholder="0.0001"
                  {...createForm.register("creditUnitPrice", { valueAsNumber: true })}
                  error={createForm.formState.errors.creditUnitPrice?.message}
                />
              )}
            </div>

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

      <Dialog open={!!editingModel} onOpenChange={(open) => !open && setEditingModel(null)}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Edit Pricing Model</DialogTitle>
          </DialogHeader>
          <form onSubmit={editForm.handleSubmit(handleUpdate)} className="space-y-4">
            <FormInput
              label="Name"
              placeholder="e.g., Standard Plan"
              {...editForm.register("name")}
              error={editForm.formState.errors.name?.message}
            />

            <div className="border rounded-lg p-4 space-y-4">
              <div className="flex items-center justify-between">
                <div>
                  <span className="font-medium">Key Access</span>
                  <p className="text-xs text-gray-11">Charge per unique key with access</p>
                </div>
                <Switch
                  checked={editForm.watch("enableKeyAccess")}
                  onCheckedChange={(checked) => editForm.setValue("enableKeyAccess", checked)}
                />
              </div>
              {editForm.watch("enableKeyAccess") && (
                <FormInput
                  label="Key Access Price (per key)"
                  type="number"
                  step="0.0001"
                  min="0"
                  placeholder="0.0001"
                  {...editForm.register("keyAccessUnitPrice", { valueAsNumber: true })}
                  error={editForm.formState.errors.keyAccessUnitPrice?.message}
                />
              )}
            </div>

            <div className="border rounded-lg p-4 space-y-4">
              <div className="flex items-center justify-between">
                <div>
                  <span className="font-medium">Verifications</span>
                  <p className="text-xs text-gray-11">Charge per API verification</p>
                </div>
                <Switch
                  checked={editForm.watch("enableVerification")}
                  onCheckedChange={(checked) => editForm.setValue("enableVerification", checked)}
                />
              </div>
              {editForm.watch("enableVerification") && (
                <FormInput
                  label="Verification Price (per unit)"
                  type="number"
                  step="0.0001"
                  min="0"
                  placeholder="0.0001"
                  {...editForm.register("verificationUnitPrice", { valueAsNumber: true })}
                  error={editForm.formState.errors.verificationUnitPrice?.message}
                />
              )}
            </div>

            <div className="border rounded-lg p-4 space-y-4">
              <div className="flex items-center justify-between">
                <div>
                  <span className="font-medium">Credits</span>
                  <p className="text-xs text-gray-11">Charge per credit</p>
                </div>
                <Switch
                  checked={editForm.watch("enableCredit")}
                  onCheckedChange={(checked) => editForm.setValue("enableCredit", checked)}
                />
              </div>
              {editForm.watch("enableCredit") && (
                <FormInput
                  label="Credit Price (per credit)"
                  type="number"
                  step="0.0001"
                  min="0"
                  placeholder="0.0001"
                  {...editForm.register("creditUnitPrice", { valueAsNumber: true })}
                  error={editForm.formState.errors.creditUnitPrice?.message}
                />
              )}
            </div>

            <p className="text-xs text-gray-11">
              Note: Updating a pricing model creates a new version. Existing invoices will use the
              previous version.
            </p>

            <DialogFooter>
              <Button type="button" variant="outline" onClick={() => setEditingModel(null)}>
                Cancel
              </Button>
              <Button type="submit" variant="primary" loading={updateMutation.isLoading}>
                Update
              </Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>
    </div>
  );
}
