const Stripe = require("stripe");

async function main() {
  const stripe = new Stripe(process.env.STRIPE_SECRET_KEY, {
    apiVersion: "2023-10-16",
    typescript: true,
  });

  await stripe.products.update("prod_Rt7pECOwzDIiHC", {
    metadata: {
      quota_requests_per_month: "250000",
      quota_logs_retention_days: "90",
      quota_audit_logs_retention_days: "90",
    },
  });

  await stripe.products.update("prod_Rt7phP6Jg6CEAX", {
    metadata: {
      quota_requests_per_month: "500000",
      quota_logs_retention_days: "90",
      quota_audit_logs_retention_days: "90",
    },
  });

  await stripe.products.update("prod_Rt7skQe4k40vB9", {
    metadata: {
      quota_requests_per_month: "1000000",
      quota_logs_retention_days: "90",
      quota_audit_logs_retention_days: "90",
    },
  });

  await stripe.products.update("prod_Rt7sxA99W66fei", {
    metadata: {
      quota_requests_per_month: "2000000",
      quota_logs_retention_days: "90",
      quota_audit_logs_retention_days: "90",
    },
  });

  await stripe.products.update("prod_Rt7sdc9yBJlixM", {
    metadata: {
      quota_requests_per_month: "10000000",
      quota_logs_retention_days: "90",
      quota_audit_logs_retention_days: "90",
    },
  });

  await stripe.products.update("prod_Rt7s7o5GODuYEc", {
    metadata: {
      quota_requests_per_month: "50000000",
      quota_logs_retention_days: "90",
      quota_audit_logs_retention_days: "90",
    },
  });

  await stripe.products.update("prod_Rt7tA07o4WOVmc", {
    metadata: {
      quota_requests_per_month: "100000000",
      quota_logs_retention_days: "90",
      quota_audit_logs_retention_days: "90",
    },
  });
}

main();
