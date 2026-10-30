export function displayNameOf(user: {
  firstName: string | null;
  lastName: string | null;
}): string | null {
  return `${user.firstName ?? ""} ${user.lastName ?? ""}`.trim() || null;
}
