import { describe, it, expect } from "vitest"
import { cn } from "@/lib/utils"

describe("cn", () => {
  it("merges multiple class strings", () => {
    // Given / When
    const result = cn("foo", "bar", "baz")
    // Then
    expect(result).toBe("foo bar baz")
  })

  it("ignores falsy values", () => {
    // Given / When
    const result = cn("foo", false, undefined, null, "", "bar")
    // Then
    expect(result).toBe("foo bar")
  })

  it("merges conflicting Tailwind classes — last one wins", () => {
    // Given — two conflicting padding utilities
    // When
    const result = cn("p-4", "p-8")
    // Then — twMerge keeps only the last
    expect(result).toBe("p-8")
  })

  it("handles conditional class objects", () => {
    // Given
    const isActive = true
    const isDisabled = false
    // When
    const result = cn("base", { active: isActive, disabled: isDisabled })
    // Then
    expect(result).toBe("base active")
  })
})
