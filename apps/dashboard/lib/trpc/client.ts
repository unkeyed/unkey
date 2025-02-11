import { createTRPCContext } from "@trpc/tanstack-react-query";
import type { Router } from "./routers";

export const {
  TRPCProvider,
  useTRPC
} = createTRPCContext<Router>();
