/**
 * Tests for EmptyState component.
 *
 * Covers: rendering, icon, title, description, action button.
 * Pure presentational — action button handles interactivity.
 */

import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { EmptyState } from "../EmptyState";

describe("EmptyState", () => {
  it("renders title and description", () => {
    render(
      <EmptyState
        title="No results"
        description="Try adjusting your filters."
      />,
    );
    expect(screen.getByRole("heading", { name: "No results" })).toBeInTheDocument();
    expect(screen.getByText("Try adjusting your filters.")).toBeInTheDocument();
  });

  it("renders icon when provided", () => {
    render(
      <EmptyState
        icon={<span data-testid="icon">📧</span>}
        title="No emails"
        description="Your inbox is empty."
      />,
    );
    expect(screen.getByTestId("icon")).toBeInTheDocument();
  });

  it("does not render icon slot when not provided", () => {
    render(
      <EmptyState
        title="Empty"
        description="Nothing here."
      />,
    );
    expect(screen.queryByTestId("icon")).not.toBeInTheDocument();
  });

  it("icon has aria-hidden", () => {
    render(
      <EmptyState
        icon={<span>📧</span>}
        title="No emails"
        description="Empty."
      />,
    );
    // The icon wrapper div has aria-hidden
    const iconWrapper = screen.getByText("📧").parentElement;
    expect(iconWrapper).toHaveAttribute("aria-hidden", "true");
  });

  it("renders action button when provided", () => {
    const handleClick = vi.fn();
    render(
      <EmptyState
        title="No jobs"
        description="Start searching."
        action={{ label: "Start Search", onClick: handleClick }}
      />,
    );
    expect(screen.getByRole("button", { name: "Start Search" })).toBeInTheDocument();
  });

  it("calls action onClick when button is clicked", async () => {
    const user = userEvent.setup();
    const handleClick = vi.fn();
    render(
      <EmptyState
        title="No jobs"
        description="Start searching."
        action={{ label: "Search", onClick: handleClick }}
      />,
    );

    await user.click(screen.getByRole("button", { name: "Search" }));
    expect(handleClick).toHaveBeenCalledTimes(1);
  });

  it("does not render action button when not provided", () => {
    render(
      <EmptyState
        title="Empty"
        description="Nothing here."
      />,
    );
    expect(screen.queryByRole("button")).not.toBeInTheDocument();
  });

  it("accepts custom className", () => {
    const { container } = render(
      <EmptyState
        title="Test"
        description="Description"
        className="custom-class"
      />,
    );
    expect(container.firstChild).toHaveAttribute("class");
    expect((container.firstChild as HTMLElement).className).toContain("custom-class");
  });
});
