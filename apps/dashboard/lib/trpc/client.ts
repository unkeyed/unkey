import { createTRPCReact } from "@trpc/react-query";
import type { Router } from "./routers";

export const trpc = createTRPCReact<Router>();
