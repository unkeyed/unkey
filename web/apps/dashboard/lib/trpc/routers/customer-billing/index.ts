import { t } from "../../trpc";
import { exportBillingData, getRevenueAnalytics, getUsageAnalytics } from "./analytics";
import { disconnectAccount, getConnectedAccount } from "./connect";
import { createEndUser, deleteEndUser, getEndUser, listEndUsers, searchEndUsers, updateEndUser, upsertEndUser } from "./end-users";
import { getInvoice, getInvoiceSummary, listInvoices } from "./invoices";
import {
  createPricingModel,
  deletePricingModel,
  getPricingModel,
  getWorkspaceCurrency,
  listPricingModels,
  updatePricingModel,
} from "./pricing-models";

export const customerBillingRouter = t.router({
  connect: t.router({
    getAccount: getConnectedAccount,
    disconnect: disconnectAccount,
  }),
  pricingModels: t.router({
    list: listPricingModels,
    get: getPricingModel,
    create: createPricingModel,
    update: updatePricingModel,
    delete: deletePricingModel,
    getWorkspaceCurrency,
  }),
  endUsers: t.router({
    list: listEndUsers,
    get: getEndUser,
    search: searchEndUsers,
    create: createEndUser,
    update: updateEndUser,
    delete: deleteEndUser,
  }),
  invoices: t.router({
    list: listInvoices,
    get: getInvoice,
    getSummary: getInvoiceSummary,
  }),
  analytics: t.router({
    revenue: getRevenueAnalytics,
    usage: getUsageAnalytics,
    export: exportBillingData,
  }),
});
