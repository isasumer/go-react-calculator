/**
 * The last line of defence: a render error anywhere below this boundary replaces the tree with a
 * plain apology and a way out, instead of an empty white page.
 *
 * It is deliberately small and stateless beyond `hasError`. There is no retry that re-renders the
 * same broken tree — if a component threw while rendering, the state that made it throw is still
 * there — so the only offer is a reload, which is also the only thing a user can do about it.
 */
import { Component, type ErrorInfo, type ReactNode } from "react";

import { Button } from "@/components/ui/button";

export interface ErrorBoundaryProps {
  readonly children: ReactNode;
}

interface ErrorBoundaryState {
  readonly hasError: boolean;
}

export class ErrorBoundary extends Component<ErrorBoundaryProps, ErrorBoundaryState> {
  override state: ErrorBoundaryState = { hasError: false };

  static getDerivedStateFromError(): ErrorBoundaryState {
    return { hasError: true };
  }

  /** React swallows the error once a boundary handles it; the console is where it stays visible. */
  override componentDidCatch(error: Error, info: ErrorInfo): void {
    console.error("Unhandled error below <ErrorBoundary>", error, info.componentStack);
  }

  private readonly handleReload = (): void => {
    window.location.reload();
  };

  override render(): ReactNode {
    if (!this.state.hasError) {
      return this.props.children;
    }

    return (
      <div
        data-ui="error-boundary"
        role="alert"
        className="grid min-h-dvh place-items-center bg-surface p-6 text-text"
      >
        <div className="w-full max-w-sm space-y-4 rounded-xl border border-border bg-surface-raised p-6 text-center shadow-sm">
          <h1 className="text-xl font-semibold tracking-tight">Something went wrong</h1>
          <p className="text-sm text-text-muted">
            The calculator stopped unexpectedly. Reloading the page usually fixes it.
          </p>
          <Button type="button" className="w-full" onClick={this.handleReload}>
            Reload the page
          </Button>
        </div>
      </div>
    );
  }
}
