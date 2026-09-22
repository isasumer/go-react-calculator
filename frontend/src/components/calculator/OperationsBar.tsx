/**
 * The row above the keypad: every operation the backend advertises that the pad itself has no key
 * for — `√`, `%` and `xʸ` today.
 *
 * It is driven by `GET /api/v1/operations` rather than by a hard-coded list, so a backend that
 * gains an operation shows up here on the next load instead of on the next frontend release. The
 * four arithmetic operations are filtered out because they already have a column of their own.
 *
 * An operation this build has no action for still renders — that is the point of discovery — but
 * disabled, and it says why in its tooltip. Pretending to support it would be worse than admitting
 * that the frontend is older than the service.
 */
import { useOperations } from "@/hooks/use-operations";
import { BINARY_OPERATIONS, UNARY_KEYS } from "@/lib/calculator-engine";
import type { BinaryOperation, UnaryKey } from "@/lib/calculator-engine";

import { Key } from "./Key";

/** Operations the keypad already owns; listing them here keeps the bar free of duplicates. */
const KEYPAD_OPERATIONS: ReadonlySet<string> = new Set(["add", "subtract", "multiply", "divide"]);

/** Spoken names. An operation we do not know is announced by its machine name, which is honest. */
const ARIA_LABELS: Readonly<Record<string, string>> = {
  sqrt: "square root",
  percent: "percent",
  power: "power",
};

/** Where the registry's symbol is not what a key should show: `^` on its own reads as a caret. */
const KEY_LABELS: Readonly<Record<string, string>> = {
  power: "xʸ",
};

const KEY_SHORTCUTS: Readonly<Record<string, string>> = {
  percent: "%",
  power: "^",
};

function isUnaryKey(name: string): name is UnaryKey {
  return (UNARY_KEYS as readonly string[]).includes(name);
}

function isBinaryOperation(name: string): name is BinaryOperation {
  return (BINARY_OPERATIONS as readonly string[]).includes(name);
}

/**
 * What pressing this operation should do, or `undefined` when this build has no action for it —
 * `√` and `%` act on the value on the display, `xʸ` selects a pending operation like any operator.
 */
function pressHandler(
  name: string,
  onUnary: (op: UnaryKey) => void,
  onOperator: (operator: BinaryOperation) => void,
): (() => void) | undefined {
  if (isUnaryKey(name)) {
    return () => {
      onUnary(name);
    };
  }
  if (isBinaryOperation(name)) {
    return () => {
      onOperator(name);
    };
  }
  return undefined;
}

export interface OperationsBarProps {
  readonly busy: boolean;
  /** `√` and `%`: act on the value already on the display. */
  readonly onUnary: (op: UnaryKey) => void;
  /** `xʸ`: selects a pending operation, exactly like a keypad operator. */
  readonly onOperator: (operator: BinaryOperation) => void;
}

export function OperationsBar({ busy, onUnary, onOperator }: OperationsBarProps) {
  const { operations } = useOperations();
  const extras = operations.filter((operation) => !KEYPAD_OPERATIONS.has(operation.name));

  return (
    <div
      data-ui="calculator.operations"
      role="group"
      aria-label="More operations"
      className="grid grid-cols-3 gap-2"
    >
      {extras.map(({ name, symbol }) => {
        const press = pressHandler(name, onUnary, onOperator);
        const unsupported = press === undefined;

        return (
          <Key
            key={name}
            label={KEY_LABELS[name] ?? symbol}
            ariaLabel={ARIA_LABELS[name] ?? name}
            variant="action"
            keyShortcut={KEY_SHORTCUTS[name]}
            disabled={busy || unsupported}
            title={unsupported ? `${name} needs a newer version of this app` : undefined}
            onPress={press}
          />
        );
      })}
    </div>
  );
}
