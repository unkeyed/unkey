export interface DiffChange {
  id: string;
  text: string;
  level: number;
  operation: string;
  path: string;
  source: string;
  section: string;
  comment?: string;
}

export interface DiffData {
  changes: DiffChange[];
}
