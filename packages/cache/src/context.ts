export interface Context {
  waitUntil: (p: Promise<unknown>) => void;
}
