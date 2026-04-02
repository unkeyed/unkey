/**
 * Thrown by `.exhaustive()` when no pattern matched the input.
 * Mirrors ts-pattern's `NonExhaustiveError`.
 */
export class NonExhaustiveError extends Error {
  constructor(public input: unknown) {
    let displayedValue: unknown;
    try {
      displayedValue = JSON.stringify(input);
    } catch {
      displayedValue = input;
    }
    super(`Pattern matching error: no pattern matches value ${displayedValue}`);
  }
}
