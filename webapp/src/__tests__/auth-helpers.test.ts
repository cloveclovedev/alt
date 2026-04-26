import { describe, it, expect, vi, beforeEach } from "vitest"
import { isAllowedUser } from "@/lib/auth-helpers"

describe("isAllowedUser", () => {
  beforeEach(() => {
    vi.stubEnv("ALLOWED_GITHUB_USER_ID", "12345")
  })

  it("allows matching numeric user ID", () => {
    expect(isAllowedUser(12345)).toBe(true)
  })

  it("allows matching string user ID", () => {
    expect(isAllowedUser("12345")).toBe(true)
  })

  it("rejects non-matching user ID", () => {
    expect(isAllowedUser(99999)).toBe(false)
  })

  it("rejects undefined", () => {
    expect(isAllowedUser(undefined)).toBe(false)
  })

  it("rejects null", () => {
    expect(isAllowedUser(null)).toBe(false)
  })
})

describe("isAllowedUser when env var is unset", () => {
  it("rejects all users when env var is empty", () => {
    vi.stubEnv("ALLOWED_GITHUB_USER_ID", "")
    expect(isAllowedUser(12345)).toBe(false)
  })

  it("rejects all users when env var is missing", () => {
    delete process.env.ALLOWED_GITHUB_USER_ID
    expect(isAllowedUser(12345)).toBe(false)
  })
})
