export type Result<TValue, TError extends { message: string } = { message: string }> =
  | {
      value: TValue;
      error?: never;
    }
  | {
      value?: never;
      error: TError;
    };

function success<TValue>(value: TValue): Result<TValue> {
  return { value };
}

function fail<TError extends { message: string }>(error: TError): Result<any, TError> {
  return { error };
}

export const result = {
  success,
  fail,
};
