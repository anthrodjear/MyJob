/**
 * Tests for utility functions (lib/utils.ts).
 *
 * Covers: cn, formatDate, formatScore, scoreLevel, truncate.
 * Pure functions — no mocking needed.
 */

import { describe, it, expect } from "vitest";
import { cn, formatDate, formatScore, scoreLevel, truncate } from "../utils";

describe("cn", () => {
  it("merges class names", () => {
    expect(cn("p-2", "p-4")).toBe("p-4");
  });

  it("handles conditional classes", () => {
    const result = cn("base", false && "hidden", "end");
    expect(result).toContain("base");
    expect(result).toContain("end");
    expect(result).not.toContain("hidden");
  });

  it("returns empty string for no inputs", () => {
    expect(cn()).toBe("");
  });
});

describe("formatDate", () => {
  it("formats a valid date string", () => {
    const result = formatDate("2026-01-15T00:00:00Z");
    expect(result).toBe("Jan 15, 2026");
  });

  it("formats a Date object", () => {
    const result = formatDate(new Date("2026-06-23T00:00:00Z"));
    expect(result).toBe("Jun 23, 2026");
  });

  it("returns — for null", () => {
    expect(formatDate(null)).toBe("—");
  });

  it("returns — for undefined", () => {
    expect(formatDate(undefined)).toBe("—");
  });

  it("returns — for invalid date string", () => {
    expect(formatDate("not-a-date")).toBe("—");
  });

  it("uses UTC to prevent timezone day shifts", () => {
    // A date that would be different day in US timezones vs UTC
    const result = formatDate("2026-01-01T03:00:00Z");
    expect(result).toBe("Jan 1, 2026");
  });
});

describe("formatScore", () => {
  it("formats a valid score", () => {
    expect(formatScore(85)).toBe("85%");
  });

  it("rounds decimal scores", () => {
    expect(formatScore(85.7)).toBe("86%");
  });

  it("clamps score to 100", () => {
    expect(formatScore(150)).toBe("100%");
  });

  it("clamps negative score to 0", () => {
    expect(formatScore(-10)).toBe("0%");
  });

  it("returns — for null", () => {
    expect(formatScore(null)).toBe("—");
  });

  it("returns — for undefined", () => {
    expect(formatScore(undefined)).toBe("—");
  });

  it("handles score of 0", () => {
    expect(formatScore(0)).toBe("0%");
  });

  it("handles score of 100", () => {
    expect(formatScore(100)).toBe("100%");
  });
});

describe("scoreLevel", () => {
  it("returns high for score >= 80", () => {
    expect(scoreLevel(80)).toBe("high");
    expect(scoreLevel(95)).toBe("high");
    expect(scoreLevel(100)).toBe("high");
  });

  it("returns medium for score 50-79", () => {
    expect(scoreLevel(50)).toBe("medium");
    expect(scoreLevel(65)).toBe("medium");
    expect(scoreLevel(79)).toBe("medium");
  });

  it("returns low for score < 50", () => {
    expect(scoreLevel(0)).toBe("low");
    expect(scoreLevel(25)).toBe("low");
    expect(scoreLevel(49)).toBe("low");
  });
});

describe("truncate", () => {
  it("returns original string if shorter than maxLen", () => {
    expect(truncate("hello", 10)).toBe("hello");
  });

  it("truncates long strings", () => {
    expect(truncate("hello world", 5)).toBe("hello...");
  });

  it("trims whitespace before truncating", () => {
    // "  hello  " → trimmed to "hello" (5 chars), equals maxLen, so no "..."
    expect(truncate("  hello  ", 5)).toBe("hello");
    // "  hello  " → trimmed to "hello" (5 chars), less than maxLen 10, no truncation
    expect(truncate("  hello  ", 10)).toBe("hello");
    // "  hello world  " → trimmed to "hello world" (11 chars), truncated to 8
    expect(truncate("  hello world  ", 8)).toBe("hello wo...");
  });

  it("returns original if equal to maxLen", () => {
    expect(truncate("hello", 5)).toBe("hello");
  });

  it("returns original if maxLen <= 0", () => {
    expect(truncate("hello", 0)).toBe("hello");
    expect(truncate("hello", -1)).toBe("hello");
  });

  it("handles empty string", () => {
    expect(truncate("", 5)).toBe("");
  });
});
