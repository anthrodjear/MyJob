/**
 * Tests for Button component.
 *
 * Covers: rendering, variants, sizes, loading state, disabled, click handler.
 * Uses @testing-library/react for DOM assertions.
 */

import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { Button } from "../Button";

describe("Button", () => {
  it("renders with text", () => {
    render(<Button>Click me</Button>);
    expect(screen.getByRole("button", { name: "Click me" })).toBeInTheDocument();
  });

  it("renders with primary variant by default", () => {
    render(<Button>Submit</Button>);
    const button = screen.getByRole("button");
    expect(button.className).toContain("bg-primary");
  });

  it("renders secondary variant", () => {
    render(<Button variant="secondary">Cancel</Button>);
    const button = screen.getByRole("button");
    expect(button.className).toContain("bg-bg-tertiary");
  });

  it("renders ghost variant", () => {
    render(<Button variant="ghost">Close</Button>);
    const button = screen.getByRole("button");
    expect(button.className).toContain("bg-transparent");
  });

  it("renders danger variant", () => {
    render(<Button variant="danger">Delete</Button>);
    const button = screen.getByRole("button");
    expect(button.className).toContain("bg-danger");
  });

  it("renders small size", () => {
    render(<Button size="sm">Small</Button>);
    const button = screen.getByRole("button");
    expect(button.className).toContain("px-3");
  });

  it("renders large size", () => {
    render(<Button size="lg">Large</Button>);
    const button = screen.getByRole("button");
    expect(button.className).toContain("px-6");
  });

  it("calls onClick when clicked", async () => {
    const user = userEvent.setup();
    const handleClick = vi.fn();
    render(<Button onClick={handleClick}>Click me</Button>);

    await user.click(screen.getByRole("button"));
    expect(handleClick).toHaveBeenCalledTimes(1);
  });

  it("is disabled when disabled prop is true", () => {
    render(<Button disabled>Disabled</Button>);
    expect(screen.getByRole("button")).toBeDisabled();
  });

  it("shows loading state with spinner", () => {
    render(<Button loading>Saving</Button>);
    const button = screen.getByRole("button");
    expect(button).toBeDisabled();
    expect(button).toHaveAttribute("aria-busy", "true");
    // Spinner SVG is present
    expect(button.querySelector("svg")).toBeInTheDocument();
  });

  it("uses custom loadingText for aria-label", () => {
    render(<Button loading loadingText="Saving…">Save</Button>);
    expect(screen.getByRole("button")).toHaveAttribute("aria-label", "Saving…");
  });

  it("defaults aria-label to Loading when loading without loadingText", () => {
    render(<Button loading>Save</Button>);
    expect(screen.getByRole("button")).toHaveAttribute("aria-label", "Loading");
  });

  it("does not show spinner when not loading", () => {
    render(<Button>Submit</Button>);
    const button = screen.getByRole("button");
    expect(button.querySelector("svg")).not.toBeInTheDocument();
  });

  it("forwards ref", () => {
    const ref = { current: null };
    render(<Button ref={ref}>With Ref</Button>);
    expect(ref.current).toBeInstanceOf(HTMLButtonElement);
  });
});
