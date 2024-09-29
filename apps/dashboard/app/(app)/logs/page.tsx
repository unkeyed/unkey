"use server";

import { generateMockLogs } from "./data";
import LogsPage from "./logs-page";

const mockLogs = generateMockLogs(50);

export default async function Page() {
  return <LogsPage logs={mockLogs} />;
}
