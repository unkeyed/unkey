import type { User } from "./types";

export type WorkOSUserProfile = {
  id: string;
  email: string;
  firstName: string | null;
  lastName: string | null;
  profilePictureUrl: string | null;
};

export function mapWorkOSUser(user: WorkOSUserProfile): User {
  return {
    id: user.id,
    email: user.email,
    firstName: user.firstName,
    lastName: user.lastName,
    avatarUrl: user.profilePictureUrl,
    fullName: user.firstName && user.lastName ? `${user.firstName} ${user.lastName}` : null,
  };
}
