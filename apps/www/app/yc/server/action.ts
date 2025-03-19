"use server";
import { PlainClient } from "@team-plain/typescript-sdk";

export type ActionError = {
  code: string;
  message: string;
};

type PlainResult = {
  error?: ActionError;
  data?: { threadId: string };
};

export type ServerResponse =
  | { status: "error"; errors: string[] }
  | { status: "success"; submitted: boolean };

export default async function create(formData: FormData): Promise<ServerResponse> {
  try {
    const result = await createPlain(formData);
    if (result.error) {
      return {
        status: "error" as const,
        errors: [result.error.message],
      };
    }
    return { status: "success" as const, submitted: true };
  } catch (e) {
    console.error("Unexpected error in form submission:", e);
    return {
      status: "error" as const,
      errors: ["An unexpected error occurred. Please try again later."],
    };
  }
}

const createPlain = async (formData: FormData): Promise<PlainResult> => {
  const client = new PlainClient({
    apiKey: process.env.PLAIN_API_KEY ?? "",
  });

  if (!client) {
    return {
      error: { code: "CONFIG_ERROR", message: "Invalid API configuration" },
    };
  }

  const name = formData.get("Full Name");
  const email = formData.get("Email");
  const ycBatch = formData.get("YC Batch");
  const workspaceId = formData.get("Workspace ID");
  const migrationFrom = formData.get("Migrating From");
  const otherDetails = formData.get("More Info");

  // if in some way the form is not valid, return an error.
  if (!name || !email || !ycBatch) {
    return {
      error: {
        code: "VALIDATION_ERROR",
        message: "All fields are required",
      },
    };
  }

  try {
    const upsertCustomerRes = await client.upsertCustomer({
      identifier: {
        emailAddress: email.toString(),
      },
      onCreate: {
        fullName: name.toString(),
        email: {
          email: email.toString(),
          isVerified: true,
        },
      },
      onUpdate: {},
    });

    if (upsertCustomerRes.error) {
      return {
        error: {
          code: "CUSTOMER_ERROR",
          message: upsertCustomerRes.error.message,
        },
      };
    }

    const createThreadRes = await client.createThread({
      customerIdentifier: {
        customerId: upsertCustomerRes.data.customer.id,
      },
      title: "Contact form",
      components: [
        {
          componentText: {
            text: `YC Batch: ${ycBatch}\nWorkspace ID: ${workspaceId}\nMigrating From: ${migrationFrom}\nMore Info: ${otherDetails}`,
          },
        },
      ],
    });

    if (createThreadRes.error) {
      return {
        error: {
          code: "THREAD_ERROR",
          message: createThreadRes.error.message,
        },
      };
    }
    return { data: { threadId: createThreadRes.data.id } };
  } catch (error) {
    console.error("Unexpected error:", error);
    return {
      error: {
        code: "UNEXPECTED_ERROR",
        message: "An unexpected error occurred while processing your request",
      },
    };
  }
};
