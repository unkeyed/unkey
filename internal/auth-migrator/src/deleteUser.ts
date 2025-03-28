import {
    WorkOS,
    RateLimitExceededException,
    WorkOSResponseError,
    User as WorkOSUser,
  } from "@workos-inc/node";
  import fs from "fs/promises";
  import path from "path";
  import pLimit from "p-limit";
  
  const workos = new WorkOS(process.env.WORKOS_API_KEY);
  const limit = pLimit(10); // Process 10 users concurrently
  
  type UserData = {
    id: string;
    email: string;
  };
  
  async function getUsers() {
    let list = await workos.userManagement.listUsers({
      limit: 100,
      order: "desc",
    });
    let users = list.data;
    let after = list.listMetadata.after;
  
    while (after) {
      list = await workos.userManagement.listUsers({
        limit: 100,
        after: after,
        order: "desc",
      });
      users = users.concat(list.data);
      after = list.listMetadata.after;
    }
    return users;
  }
  
  async function deleteUser(userId: string): Promise<void> {
    try {
      await workos.userManagement.deleteUser(userId);
      console.log(`Successfully deleted user ${userId}`);
    } catch (error) {
      if (error instanceof RateLimitExceededException) {
        console.error(`Rate limit exceeded for user ${userId}, retrying...`);
        // Wait and retry once
        await new Promise((resolve) => setTimeout(resolve, 1000));
        return deleteUser(userId);
      } else if (error && typeof error === "object" && "errors" in error) {
        const workosError = error as WorkOSResponseError;
        const errorMessage = workosError.errors;
        console.error(`Failed to delete user ${userId}: ${errorMessage}`);
      } else {
        const errorMessage =
          error instanceof Error ? error.message : "Unknown error";
        console.error(
          `Unexpected error deleting user ${userId}: ${errorMessage}`
        );
      }
    }
  }
  
  export async function processUsers(): Promise<void> {
    try {
      const users = await getUsers();
      console.log(`Processing ${users.length} users...`);
  
      const results = await Promise.allSettled(
        users.map((user) => limit(() => deleteUser(user.id)))
      );
  
      const successful = results.filter((r) => r.status === "fulfilled").length;
      const failed = results.filter((r) => r.status === "rejected").length;
  
      console.log(
        `Completed processing users:\n- Successfully deleted: ${successful}\n- Failed: ${failed}`
      );
    } catch (error) {
      console.error("Failed to process users:", error);
      throw error;
    }
  }
  
  processUsers().then(() => process.exit(0));