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

// rome-ignore lint/suspicious/noExplicitAny: <explanation>
function fail<TError extends { message: string }>(error: TError): Result<any, TError> {
  return { error };
}

export const result = {
  success,
  fail,
};
