// Returns the earlier of `sourceExpires` and `gracePeriodEnd`. When the
// source key has no expiry, returns `gracePeriodEnd` unchanged. Encodes the
// invariant that key rotation must never extend a key's lifetime past the
// expiry that was originally configured — a 24-hour grace period applied to
// a key that expires in 5 minutes still revokes after 5 minutes.
export function capGracePeriodAtSourceExpiry(
  sourceExpires: Date | null,
  gracePeriodEnd: Date,
): Date {
  if (sourceExpires && sourceExpires.getTime() < gracePeriodEnd.getTime()) {
    return sourceExpires;
  }
  return gracePeriodEnd;
}
