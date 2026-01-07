import type { Column } from "@/components/virtual-table/types";

export type VerificationLog = {
  request_id: string;
  time: number;
  outcome: string;
  region?: string;
  tags?: string[];
  [key: string]: any;
};

export type DataHook<TLog extends VerificationLog> = () => {
  realtimeLogs: TLog[];
  historicalLogs: TLog[];
  isLoading: boolean;
  isLoadingMore: boolean;
  loadMore: () => void;
  hasMore: boolean;
  totalCount: number;
};

export type EmptyStateConfig = {
  title: string;
  description: string;
  actions?: React.ReactNode;
};