/**
 * RejectDialog — modal dialog for rejecting an approval request.
 *
 * Requires a rejection reason before confirming. Uses focus trap,
 * escape key handling, and body scroll lock.
 *
 * @example
 *   <RejectDialog isOpen={true} onConfirm={handleReject} onCancel={handleClose} />
 */

"use client";

import { useState, useCallback, useEffect, useRef } from "react";
import { Button } from "@/components/shared/Button";

interface RejectDialogProps {
  /** Whether the dialog is open. */
  isOpen: boolean;
  /** Callback when rejection is confirmed with a reason. */
  onConfirm: (reason: string) => void;
  /** Callback when the dialog is cancelled. */
  onCancel: () => void;
}

/** Focusable elements selector for focus trap. */
const FOCUSABLE = 'button, [href], input, select, textarea, [tabindex]:not([tabindex="-1"])';

/**
 * RejectDialog — rejection confirmation with required reason.
 *
 * Accessibility:
 * - Focus trap within dialog (Tab/Shift+Tab cycles)
 * - Escape key closes dialog
 * - `role="dialog"` with `aria-modal="true"`
 * - `aria-labelledby` for dialog title
 * - Body scroll lock when open
 * - Backdrop click closes dialog
 */
export function RejectDialog({ isOpen, onConfirm, onCancel }: RejectDialogProps) {
  const [reason, setReason] = useState("");
  const dialogRef = useRef<HTMLDivElement>(null);
  const inputRef = useRef<HTMLTextAreaElement>(null);
  const previousFocusRef = useRef<HTMLElement | null>(null);
  const titleId = "reject-dialog-title";

  // Store the element that had focus before the dialog opened
  useEffect(() => {
    if (isOpen) {
      previousFocusRef.current = document.activeElement as HTMLElement;
      // Lock body scroll
      document.body.style.overflow = "hidden";
      // Focus the textarea
      setTimeout(() => inputRef.current?.focus(), 0);
    }
    return () => {
      document.body.style.overflow = "";
      // Restore focus to the element that triggered the dialog
      previousFocusRef.current?.focus();
    };
  }, [isOpen]);

  // Focus trap + Escape key handler
  useEffect(() => {
    if (!isOpen) return;

    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === "Escape") {
        onCancel();
        return;
      }

      // Focus trap
      if (e.key === "Tab" && dialogRef.current) {
        const focusable = dialogRef.current.querySelectorAll(FOCUSABLE);
        if (focusable.length === 0) return;

        const first = focusable[0] as HTMLElement;
        const last = focusable[focusable.length - 1] as HTMLElement;

        if (e.shiftKey) {
          if (document.activeElement === first) {
            e.preventDefault();
            last.focus();
          }
        } else {
          if (document.activeElement === last) {
            e.preventDefault();
            first.focus();
          }
        }
      }
    };

    document.addEventListener("keydown", handleKeyDown);
    return () => document.removeEventListener("keydown", handleKeyDown);
  }, [isOpen, onCancel]);

  const handleSubmit = useCallback(
    (e: React.FormEvent) => {
      e.preventDefault();
      const trimmed = reason.trim();
      if (trimmed) {
        onConfirm(trimmed);
        setReason("");
      }
    },
    [reason, onConfirm],
  );

  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      {/* Backdrop */}
      <div
        className="absolute inset-0 bg-black/50"
        onClick={onCancel}
        aria-hidden="true"
      />

      {/* Dialog */}
      <div
        ref={dialogRef}
        role="dialog"
        aria-modal="true"
        aria-labelledby={titleId}
        className="relative mx-4 w-full max-w-md rounded-lg border border-border bg-bg-secondary p-6 shadow-lg"
      >
        <h2 id={titleId} className="mb-4 text-lg font-semibold text-text-primary">
          Reject Application
        </h2>

        <form onSubmit={handleSubmit}>
          <label htmlFor="reject-reason" className="mb-2 block text-sm text-text-secondary">
            Reason for rejection <span className="text-danger">*</span>
          </label>
          <textarea
            ref={inputRef}
            id="reject-reason"
            className="w-full rounded-md border border-border bg-bg-primary p-3 text-sm text-text-primary placeholder:text-text-tertiary focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
            rows={3}
            value={reason}
            onChange={(e) => setReason(e.target.value)}
            placeholder="Explain why this application is being rejected..."
            required
            aria-describedby="reject-reason-hint"
          />
          <p id="reject-reason-hint" className="mt-1 text-xs text-text-tertiary">
            This reason will be stored in the audit trail.
          </p>

          <div className="mt-4 flex justify-end gap-2">
            <Button type="button" variant="secondary" onClick={onCancel}>
              Cancel
            </Button>
            <Button
              type="submit"
              variant="danger"
              disabled={!reason.trim()}
            >
              Reject
            </Button>
          </div>
        </form>
      </div>
    </div>
  );
}
