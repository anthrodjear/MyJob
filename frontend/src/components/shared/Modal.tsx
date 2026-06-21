/**
 * Modal — accessible dialog overlay.
 *
 * Provides:
 * - Focus trap (Tab/Shift+Tab cycle within modal)
 * - Scroll locking (body scroll disabled when open)
 * - Escape key dismissal
 * - Backdrop click dismissal
 * - `role="dialog"` + `aria-modal="true"` + `aria-label`
 * - Focus restoration on close
 *
 * Uses React 19 ref-as-prop pattern.
 *
 * @example
 *   <Modal open={show} onClose={() => setShow(false)} title="Confirm">
 *     <p>Are you sure?</p>
 *   </Modal>
 */

"use client";

import { type ReactNode, useId, useEffect, useRef } from "react";
import { cn } from "@/lib/utils";
import { X } from "lucide-react";

interface ModalProps {
  /** Whether the modal is visible. */
  open: boolean;
  /** Called when the modal should close (Escape, backdrop click, close button). */
  onClose: () => void;
  /** Optional title — also used as `aria-label` for screen readers. */
  title?: string;
  /** Modal body content. */
  children: ReactNode;
  /** Additional CSS classes for the dialog panel. */
  className?: string;
}

/**
 * Modal — accessible dialog overlay.
 *
 * Accessibility:
 * - `role="dialog"` + `aria-modal="true"` for screen reader semantics
 * - Focus trap: Tab cycles through focusable elements inside the modal
 * - Escape key closes the modal
 * - Backdrop click closes the modal
 * - Body scroll is locked while open
 * - Focus is restored to the previously focused element on close
 *
 * @see https://react-aria.adobe.com/Modal/useModalOverlay.html
 */
export function Modal({
  open,
  onClose,
  title,
  children,
  className,
}: ModalProps) {
  const panelRef = useRef<HTMLDivElement>(null);
  const previousFocusRef = useRef<HTMLElement | null>(null);
  const titleId = useId();

  // Store the previously focused element and manage focus trap
  useEffect(() => {
    if (!open) return;

    // Remember what was focused before opening
    previousFocusRef.current = document.activeElement as HTMLElement;

    // Move focus into the modal panel
    const panel = panelRef.current;
    if (panel != null) {
      const focusable = panel.querySelector<HTMLElement>(
        "button, [href], input, select, textarea, [tabindex]:not([tabindex='-1'])",
      );
      focusable?.focus();
    }

    return () => {
      // Restore focus on unmount
      previousFocusRef.current?.focus();
    };
  }, [open]);

  // Escape key + body scroll lock
  useEffect(() => {
    if (!open) return;

    function handleKeyDown(e: KeyboardEvent) {
      if (e.key === "Escape") {
        onClose();
        return;
      }

      // Focus trap: Tab/Shift+Tab cycles within the modal
      if (e.key === "Tab") {
        const panel = panelRef.current;
        if (panel == null) return;

        const focusable = panel.querySelectorAll<HTMLElement>(
          "button, [href], input, select, textarea, [tabindex]:not([tabindex='-1'])",
        );
        if (focusable.length === 0) return;

        const first = focusable[0];
        const last = focusable[focusable.length - 1];

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
    }

    document.addEventListener("keydown", handleKeyDown);
    document.body.style.overflow = "hidden";

    return () => {
      document.removeEventListener("keydown", handleKeyDown);
      document.body.style.overflow = "";
    };
  }, [open, onClose]);

  if (!open) return null;

  return (
    <div
      className="fixed inset-0 z-[--z-modal] flex items-center justify-center"
      role="dialog"
      aria-modal="true"
      aria-labelledby={title != null ? titleId : undefined}
      aria-label={title == null ? "Dialog" : undefined}
    >
      {/* Backdrop */}
      <div
        className="fixed inset-0 bg-black/50"
        onClick={onClose}
        aria-hidden="true"
      />

      {/* Dialog panel */}
      <div
        ref={panelRef}
        className={cn(
          "relative z-10 w-full max-w-lg rounded-xl bg-surface p-6 shadow-lg",
          className,
        )}
      >
        {title != null && (
          <div className="mb-4 flex items-center justify-between">
            <h2 id={titleId} className="text-lg font-semibold text-text-primary">
              {title}
            </h2>
            <button
              type="button"
              onClick={onClose}
              className="rounded-md p-1 text-text-tertiary hover:text-text-primary focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary"
              aria-label="Close"
            >
              <X className="h-5 w-5" />
            </button>
          </div>
        )}
        {children}
      </div>
    </div>
  );
}
