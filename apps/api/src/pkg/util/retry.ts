export function retry<T>(attempts: number, fn: () => T): T {
  let err: Error | undefined = undefined;
  for (let i = attempts; i >= 0; i--) {
    try {
      return fn();
    } catch (e) {
      console.warn(e);
      err = e as Error;
    }
  }
  throw err;
}
