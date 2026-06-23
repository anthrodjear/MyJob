/**
 * Tests for Badge component.
 *
 * Covers: rendering, variants, content.
 * Pure presentational — no client-side behavior.
 */

import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import { Badge } from "../Badge";

describe("Badge", () => {
  it("renders with text", () => {
    render(<Badge>Active</Badge>);
    expect(screen.getByText("Active")).toBeInTheDocument();
  });

  it("renders as inline span element", () => {
    render(<Badge>Test</Badge>);
    const badge = screen.getByText("Test");
    expect(badge.tagName).toBe("SPAN");
  });

  it("applies default variant styles", () => {
    render(<Badge>Default</Badge>);
    const badge = screen.getByText("Default");
    expect(badge.className).toContain("bg-bg-tertiary");
  });

  it("applies success variant styles", () => {
    render(<Badge variant="success">Passed</Badge>);
    const badge = screen.getByText("Passed");
    expect(badge.className).toContain("bg-success-light");
  });

  it("applies warning variant styles", () => {
    render(<Badge variant="warning">Pending</Badge>);
    const badge = screen.getByText("Pending");
    expect(badge.className).toContain("bg-warning-light");
  });

  it("applies danger variant styles", () => {
    render(<Badge variant="danger">Failed</Badge>);
    const badge = screen.getByText("Failed");
    expect(badge.className).toContain("bg-danger-light");
  });

  it("applies info variant styles", () => {
    render(<Badge variant="info">Info</Badge>);
    const badge = screen.getByText("Info");
    expect(badge.className).toContain("bg-info-light");
  });

  it("accepts custom className", () => {
    render(<Badge className="ml-2">Custom</Badge>);
    const badge = screen.getByText("Custom");
    expect(badge.className).toContain("ml-2");
    // Still has base styles
    expect(badge.className).toContain("rounded-full");
  });

  it("has base badge styles", () => {
    render(<Badge>Base</Badge>);
    const badge = screen.getByText("Base");
    expect(badge.className).toContain("text-xs");
    expect(badge.className).toContain("font-medium");
  });
});
