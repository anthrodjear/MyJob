/**
 * ToastProvider — global toast notifications.
 *
 * Provides a `toast(message, type?)` function via context.
 * Renders toasts in a fixed bottom-right stack with auto-dismiss.
 *
 * @example
 *   <ToastProvider>
 *     <App />
 *   </ToastProvider>
 *
 *   // In any component:
 *   const { toast } = useToast();
 *   toast("Job saved", "success");
 *   toast("Failed to load", "error");
 */

"use client";

import { createContext, useContext, useState, useCallback, useRef } from "react";
import { cn } from "@/lib/utils";
import { X } from "lucide-react";

type ToastType = "success" | "error" | "info";

interface Toast {
  id: string;
  message: string;
  type: ToastType;
}

interface ToastContextValue {
  toast: (message: string, type?: ToastType) => void;
}

const ToastContext = createContext<ToastContextValue>({
  toast: () => {},
});

/**
 * Hook to access toast function.
 * Must be used within ToastProvider.
 */
export function useToast() {
  return useContext(ToastContext);
}

// Type styles defined outside component — stable reference across renders
const TYPE_STYLES: Record<ToastType, string> = {
  success: "bg-success text-text-inverse",
  error: "bg-danger text-text-inverse",
  info: "bg-primary text-text-inverse",
};

// Maximum concurrent toasts — prevents DoS from toast loops
const MAX_TOASTS = 5;

/**
 * ToastProvider — global toast notifications.
 *
 * Features:
 * - Bottom-right stacked toasts (max 5)
 * - Auto-dismiss after 5 seconds
 * - Manual dismiss via close button
 * - 3 types: success (green), error (red), info (blue)
 * - `role="alert"` for screen reader announcements
 * - Tracks timeouts to clear on manual dismiss (no orphaned timers)
 *
 * Accessibility:
 * - Each toast has `role="alert"` (live region)
 * - Close button has `aria-label="Dismiss"`
 * - Icon is decorative (`aria-hidden="true"`)
 */
export function ToastProvider({
  children,
}: {
  children: React.ReactNode;
}) {
  const [toasts, setToasts] = useState<Toast[]>([]);
  const timeoutsRef = useRef<Map<string, ReturnType<typeof setTimeout>>>(new Map());

  const toast = useCallback((message: string, type: ToastType = "info") => {
    const id = crypto.randomUUID();
    setToasts((prev) => {
      const next = [...prev, { id, message, type }];
      // Cap at MAX_TOASTS — keep newest
      return next.slice(-MAX_TOASTS);
    });

    // Auto-dismiss after 5 seconds
    const timeout = setTimeout(() => {
      dismiss(id);
    }, 5_000);
    timeoutsRef.current.set(id, timeout);
  }, []);

  const dismiss = useCallback((id: string) => {
    // Clear associated timeout if exists
    const timeout = timeoutsRef.current.get(id);
    if (timeout != null) {
      clearTimeout(timeout);
      timeoutsRef.current.delete(id);
    }
    setToasts((prev) => prev.filter((t) => t.id !== id));
  }, []);

  return (
    <ToastContext.Provider value={{ toast }}>
      {children}
      <div className="fixed bottom-4 right-4 z-[--z-toast] flex flex-col gap-2">
        {toasts.map((t) => (
          <div
            key={t.id}
            className={cn(
              "flex items-center gap-3 rounded-lg px-4 py-3 text-sm shadow-lg",
              TYPE_STYLES[t.type],
            )}
            role="alert"
          >
            <span>{t.message}</span>
            <button
              onClick={() => dismiss(t.id)}
              className="ml-2 opacity-70 hover:opacity-100 focus:opacity-100"
              aria-label="Dismiss"
            >
              <X className="h-4 w-4" aria-hidden="true" />
            </button>
          </div>
        ))}
      </div>
    </ToastContext.Provider>
  );
}
