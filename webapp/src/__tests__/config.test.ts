import { describe, it, expect } from "vitest"
import { castValueByType } from "@/lib/config"

describe("castValueByType", () => {
  it("returns string as-is for type=string", () => {
    expect(castValueByType("hello", "string")).toBe("hello")
  })

  it("parses number for type=number", () => {
    expect(castValueByType("42", "number")).toBe(42)
    expect(castValueByType("3.14", "number")).toBe(3.14)
  })

  it("throws on invalid number", () => {
    expect(() => castValueByType("not-a-number", "number")).toThrow()
  })

  it("returns true/false for type=boolean", () => {
    expect(castValueByType("true", "boolean")).toBe(true)
    expect(castValueByType("false", "boolean")).toBe(false)
    expect(castValueByType(true, "boolean")).toBe(true)
  })

  it("parses JSON for type=array", () => {
    expect(castValueByType('["a","b"]', "array")).toEqual(["a", "b"])
  })

  it("throws when array JSON is not an array", () => {
    expect(() => castValueByType('{"a":1}', "array")).toThrow()
  })

  it("parses JSON for type=object", () => {
    expect(castValueByType('{"a":1}', "object")).toEqual({ a: 1 })
  })

  it("throws when object JSON is not an object", () => {
    expect(() => castValueByType('[1,2]', "object")).toThrow()
  })

  it("falls back to identity for unknown type", () => {
    expect(castValueByType("x", "unknown")).toBe("x")
  })
})
