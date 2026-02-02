"use client";

import { PageLoading } from "@/components/dashboard/page-loading";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { trpc } from "@/lib/trpc/client";
import { Eye, Page2 } from "@unkey/icons";
import {
  Button,
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  Empty,
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@unkey/ui";
import { useState } from "react";
import { BillingNavbar } from "../billing-navbar";

type InvoiceStatus = "draft" | "open" | "paid" | "void" | "uncollectible";

const STATUS_COLORS: Record<InvoiceStatus, string> = {
  draft: "bg-gray-3 text-gray-11",
  open: "bg-warning-3 text-warning-11",
  paid: "bg-success-3 text-success-11",
  void: "bg-gray-3 text-gray-11",
  uncollectible: "bg-error-3 text-error-11",
};

interface InvoiceItem {
  id: string;
  endUserId: string;
  stripeInvoiceId: string;
  billingPeriodStart: number;
  billingPeriodEnd: number;
  verificationCount: number;
  ratelimitCount: number;
  totalAmount: number;
  currency: string;
  status: string;
  createdAtM: number;
  endUser?: {
    externalId: string;
    email?: string | null;
  } | null;
}

interface TransactionItem {
  id: string;
  amount: number;
  status: string;
}

export default function InvoicesPage() {
  const workspace = useWorkspaceNavigation();
  const [statusFilter, setStatusFilter] = useState<InvoiceStatus | "all">("all");
  const [viewingInvoice, setViewingInvoice] = useState<string | null>(null);

  const { data: invoices, isLoading } = trpc.customerBilling.invoices.list.useQuery({
    status: statusFilter === "all" ? undefined : statusFilter,
    limit: 100,
  });

  const { data: summary } = trpc.customerBilling.invoices.getSummary.useQuery();

  const { data: invoiceDetails, isLoading: isLoadingDetails } =
    trpc.customerBilling.invoices.get.useQuery(
      { id: viewingInvoice ?? "" },
      { enabled: !!viewingInvoice },
    );

  // Check if billing beta is enabled
  if (!workspace.betaFeatures.billing) {
    return (
      <div>
        <BillingNavbar activePage={{ href: "invoices", text: "Invoices" }} />
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
    return <PageLoading message="Loading invoices..." />;
  }

  return (
    <div>
      <BillingNavbar activePage={{ href: "invoices", text: "Invoices" }} />
      <div className="py-3 w-full flex items-center justify-center">
        <div className="w-[900px] flex flex-col justify-center items-center gap-5 mx-6 mt-4">
          {/* Summary Cards */}
          {summary && (
            <div className="w-full grid grid-cols-4 gap-4">
              <div className="p-4 border rounded-lg">
                <div className="text-sm text-gray-11">Total Revenue</div>
                <div className="text-2xl font-semibold">
                  ${(summary.totalRevenue / 100).toFixed(2)}
                </div>
              </div>
              <div className="p-4 border rounded-lg">
                <div className="text-sm text-gray-11">Pending</div>
                <div className="text-2xl font-semibold">
                  ${(summary.pendingRevenue / 100).toFixed(2)}
                </div>
              </div>
              <div className="p-4 border rounded-lg">
                <div className="text-sm text-gray-11">Total Invoices</div>
                <div className="text-2xl font-semibold">{summary.totalInvoices}</div>
              </div>
              <div className="p-4 border rounded-lg">
                <div className="text-sm text-gray-11">Paid</div>
                <div className="text-2xl font-semibold">{summary.statusCounts.paid}</div>
              </div>
            </div>
          )}

          {/* Filters */}
          <div className="w-full flex justify-between items-center">
            <h2 className="text-lg font-medium">Invoices</h2>
            <div className="flex items-center gap-2">
              <span className="text-sm text-gray-11">Status:</span>
              <Select
                value={statusFilter}
                onValueChange={(value) => setStatusFilter(value as InvoiceStatus | "all")}
              >
                <SelectTrigger className="w-[150px]">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="all">All</SelectItem>
                  <SelectItem value="draft">Draft</SelectItem>
                  <SelectItem value="open">Open</SelectItem>
                  <SelectItem value="paid">Paid</SelectItem>
                  <SelectItem value="void">Void</SelectItem>
                  <SelectItem value="uncollectible">Uncollectible</SelectItem>
                </SelectContent>
              </Select>
            </div>
          </div>

          {/* Invoice List */}
          {invoices && invoices.length > 0 ? (
            <div className="w-full border rounded-lg overflow-hidden">
              <table className="w-full">
                <thead className="bg-gray-2">
                  <tr>
                    <th className="px-4 py-3 text-left text-sm font-medium text-gray-11">
                      Invoice
                    </th>
                    <th className="px-4 py-3 text-left text-sm font-medium text-gray-11">
                      End User
                    </th>
                    <th className="px-4 py-3 text-left text-sm font-medium text-gray-11">Period</th>
                    <th className="px-4 py-3 text-right text-sm font-medium text-gray-11">
                      Amount
                    </th>
                    <th className="px-4 py-3 text-center text-sm font-medium text-gray-11">
                      Status
                    </th>
                    <th className="px-4 py-3 text-right text-sm font-medium text-gray-11">
                      Actions
                    </th>
                  </tr>
                </thead>
                <tbody className="divide-y">
                  {invoices.map((invoice: InvoiceItem) => (
                    <tr key={invoice.id} className="hover:bg-gray-2">
                      <td className="px-4 py-3">
                        <div className="flex items-center gap-2">
                          <Page2 className="w-4 h-4 text-gray-11" />
                          <span className="font-mono text-sm">{invoice.id.slice(0, 12)}...</span>
                        </div>
                      </td>
                      <td className="px-4 py-3">
                        <span className="text-sm">{invoice.endUser?.externalId ?? "-"}</span>
                      </td>
                      <td className="px-4 py-3">
                        <span className="text-sm text-gray-11">
                          {new Date(invoice.billingPeriodStart).toLocaleDateString()} -{" "}
                          {new Date(invoice.billingPeriodEnd).toLocaleDateString()}
                        </span>
                      </td>
                      <td className="px-4 py-3 text-right">
                        <span className="font-medium">
                          ${(invoice.totalAmount / 100).toFixed(2)} {invoice.currency}
                        </span>
                      </td>
                      <td className="px-4 py-3 text-center">
                        <span
                          className={`px-2 py-1 rounded text-xs ${STATUS_COLORS[invoice.status as InvoiceStatus]}`}
                        >
                          {invoice.status}
                        </span>
                      </td>
                      <td className="px-4 py-3 text-right">
                        <Button
                          variant="ghost"
                          size="sm"
                          onClick={() => setViewingInvoice(invoice.id)}
                        >
                          <Eye className="w-4 h-4" />
                        </Button>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          ) : (
            <Empty className="w-full">
              <Empty.Icon />
              <Empty.Title>No Invoices</Empty.Title>
              <Empty.Description>
                Invoices will appear here once they are generated for your end users.
              </Empty.Description>
            </Empty>
          )}
        </div>
      </div>

      {/* Invoice Detail Dialog */}
      <Dialog open={!!viewingInvoice} onOpenChange={(open) => !open && setViewingInvoice(null)}>
        <DialogContent className="max-w-2xl">
          <DialogHeader>
            <DialogTitle>Invoice Details</DialogTitle>
          </DialogHeader>
          {isLoadingDetails ? (
            <div className="py-8 text-center text-gray-11">Loading...</div>
          ) : invoiceDetails ? (
            <div className="space-y-6">
              {/* Invoice Header */}
              <div className="flex justify-between items-start">
                <div>
                  <p className="font-mono text-sm text-gray-11">{invoiceDetails.id}</p>
                  <p className="text-2xl font-semibold mt-1">
                    ${(invoiceDetails.totalAmount / 100).toFixed(2)} {invoiceDetails.currency}
                  </p>
                </div>
                <span
                  className={`px-3 py-1 rounded ${STATUS_COLORS[invoiceDetails.status as InvoiceStatus]}`}
                >
                  {invoiceDetails.status}
                </span>
              </div>

              {/* Billing Period */}
              <div className="grid grid-cols-2 gap-4 p-4 bg-gray-2 rounded-lg">
                <div>
                  <span className="text-sm text-gray-11">Billing Period Start</span>
                  <p>{new Date(invoiceDetails.billingPeriodStart).toLocaleDateString()}</p>
                </div>
                <div>
                  <span className="text-sm text-gray-11">Billing Period End</span>
                  <p>{new Date(invoiceDetails.billingPeriodEnd).toLocaleDateString()}</p>
                </div>
              </div>

              {/* End User Info */}
              {invoiceDetails.endUser && (
                <div className="p-4 border rounded-lg">
                  <h4 className="text-sm font-medium mb-2">End User</h4>
                  <div className="grid grid-cols-2 gap-2 text-sm">
                    <div>
                      <span className="text-gray-11">External ID: </span>
                      <span className="font-mono">{invoiceDetails.endUser.externalId}</span>
                    </div>
                    {invoiceDetails.endUser.email && (
                      <div>
                        <span className="text-gray-11">Email: </span>
                        <span>{invoiceDetails.endUser.email}</span>
                      </div>
                    )}
                    {invoiceDetails.endUser.pricingModel && (
                      <div>
                        <span className="text-gray-11">Pricing Model: </span>
                        <span>{invoiceDetails.endUser.pricingModel.name}</span>
                      </div>
                    )}
                  </div>
                </div>
              )}

              {/* Usage Breakdown */}
              <div className="p-4 border rounded-lg">
                <h4 className="text-sm font-medium mb-2">Usage Breakdown</h4>
                <div className="space-y-2">
                  <div className="flex justify-between text-sm">
                    <span>Verifications</span>
                    <span>{invoiceDetails.verificationCount.toLocaleString()}</span>
                  </div>
                  <div className="flex justify-between text-sm">
                    <span>Rate Limits</span>
                    <span>{invoiceDetails.ratelimitCount.toLocaleString()}</span>
                  </div>
                </div>
              </div>

              {/* Transactions */}
              {invoiceDetails.transactions && invoiceDetails.transactions.length > 0 && (
                <div className="p-4 border rounded-lg">
                  <h4 className="text-sm font-medium mb-2">Payment History</h4>
                  <div className="space-y-2">
                    {invoiceDetails.transactions.map((tx: TransactionItem) => (
                      <div key={tx.id} className="flex justify-between items-center text-sm">
                        <div>
                          <span className="font-mono text-xs text-gray-11">
                            {tx.id.slice(0, 12)}...
                          </span>
                          <span className="ml-2">${(tx.amount / 100).toFixed(2)}</span>
                        </div>
                        <span
                          className={`px-2 py-0.5 rounded text-xs ${
                            tx.status === "succeeded"
                              ? "bg-success-3 text-success-11"
                              : tx.status === "failed"
                                ? "bg-error-3 text-error-11"
                                : "bg-warning-3 text-warning-11"
                          }`}
                        >
                          {tx.status}
                        </span>
                      </div>
                    ))}
                  </div>
                </div>
              )}

              {/* Stripe Link */}
              <div className="text-sm text-gray-11">
                <span>Stripe Invoice ID: </span>
                <span className="font-mono">{invoiceDetails.stripeInvoiceId}</span>
              </div>
            </div>
          ) : null}
          <DialogFooter>
            <Button variant="outline" onClick={() => setViewingInvoice(null)}>
              Close
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
