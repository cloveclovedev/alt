export function isAllowedUser(
  profileId: string | number | null | undefined
): boolean {
  if (profileId == null) return false
  return String(profileId) === process.env.ALLOWED_GITHUB_USER_ID
}
