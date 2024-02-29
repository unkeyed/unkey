type BaseError = {
  message: string;
};

export type Result<TValue, TError extends BaseError = BaseError> =
  | {
      value: TValue;
      error?: never;
    }
  | {
      value?: never;
      error: TError;
    };

function success(): Result<void, any>;
function success<TValue, TError extends BaseError = BaseError>(
  value: TValue,
): Result<TValue, TError>;
function success<TValue, TError extends BaseError = BaseError>(
  value?: TValue,
): Result<TValue, TError> {
  // @ts-expect-error
  return { value };
}

function fail<TError extends BaseError>(error: TError): Result<any, TError> {
  return { error };
}

export const result = {
  success,
  fail,
};
