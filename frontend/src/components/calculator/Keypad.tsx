/**
 * The 4×5 pad, in the order a hand expects it:
 *
 * ```
 * AC  CE  ⌫  ÷
 *  7   8  9  ×
 *  4   5  6  −
 *  1   2  3  +
 *  ±   0  .  =
 * ```
 *
 * The fourth column is the operator column and carries the accent; `=` closes it. Every key is
 * described here as data rather than as markup so the order, the labels and the keyboard
 * equivalents can be read in one place — and so the busy rule ("everything except `AC`") is one
 * expression rather than twenty attributes.
 */
import type { BinaryOperation, Digit } from "@/lib/calculator-engine";

import { Key, type KeyVariant } from "./Key";

export interface KeypadProps {
  /** While true every key except `AC` is disabled: the store ignores them anyway (ADR-0008). */
  readonly busy: boolean;
  readonly onDigit: (digit: Digit) => void;
  readonly onDecimal: () => void;
  readonly onToggleSign: () => void;
  readonly onBackspace: () => void;
  readonly onClearEntry: () => void;
  readonly onClearAll: () => void;
  readonly onOperator: (operator: BinaryOperation) => void;
  readonly onEquals: () => void;
}

interface KeyDescriptor {
  readonly id: string;
  readonly label: string;
  readonly ariaLabel: string;
  readonly variant: KeyVariant;
  readonly press: () => void;
  readonly keyShortcut?: string;
  /** `AC` stays live while a calculation is pending: it is the way out of a slow request. */
  readonly ignoresBusy?: boolean;
}

/**
 * The three digit rows with the operator that ends each of them. Digits read as themselves — a
 * screen reader saying "7" is exactly what the key means — while symbols need a spoken label.
 */
const NUMBER_ROWS = [
  { digits: ["7", "8", "9"], operation: "multiply", symbol: "×", name: "multiply", shortcut: "*" },
  { digits: ["4", "5", "6"], operation: "subtract", symbol: "−", name: "subtract", shortcut: "-" },
  { digits: ["1", "2", "3"], operation: "add", symbol: "+", name: "add", shortcut: "+" },
] as const satisfies readonly {
  digits: readonly Digit[];
  operation: BinaryOperation;
  symbol: string;
  name: string;
  shortcut: string;
}[];

function buildKeys(props: KeypadProps): readonly KeyDescriptor[] {
  const { onDigit, onOperator } = props;

  const digitKey = (digit: Digit): KeyDescriptor => ({
    id: digit,
    label: digit,
    ariaLabel: digit,
    variant: "digit",
    keyShortcut: digit,
    press: () => {
      onDigit(digit);
    },
  });

  return [
    {
      id: "clear-all",
      label: "AC",
      ariaLabel: "clear all",
      variant: "action",
      keyShortcut: "Escape",
      ignoresBusy: true,
      press: props.onClearAll,
    },
    {
      id: "clear-entry",
      label: "CE",
      ariaLabel: "clear entry",
      variant: "action",
      keyShortcut: "Delete",
      press: props.onClearEntry,
    },
    {
      id: "backspace",
      label: "⌫",
      ariaLabel: "backspace",
      variant: "action",
      keyShortcut: "Backspace",
      press: props.onBackspace,
    },
    {
      id: "divide",
      label: "÷",
      ariaLabel: "divide",
      variant: "operator",
      keyShortcut: "/",
      press: () => {
        onOperator("divide");
      },
    },
    ...NUMBER_ROWS.flatMap((row) => [
      ...row.digits.map(digitKey),
      {
        id: row.operation,
        label: row.symbol,
        ariaLabel: row.name,
        variant: "operator",
        keyShortcut: row.shortcut,
        press: () => {
          onOperator(row.operation);
        },
      } satisfies KeyDescriptor,
    ]),
    {
      id: "toggle-sign",
      label: "±",
      ariaLabel: "toggle sign",
      variant: "action",
      press: props.onToggleSign,
    },
    digitKey("0"),
    {
      id: "decimal",
      label: ".",
      ariaLabel: "decimal point",
      variant: "action",
      keyShortcut: ".",
      press: props.onDecimal,
    },
    {
      id: "equals",
      label: "=",
      ariaLabel: "equals",
      variant: "equals",
      keyShortcut: "Enter",
      press: props.onEquals,
    },
  ];
}

export function Keypad(props: KeypadProps) {
  return (
    <div
      data-ui="calculator.keypad"
      role="group"
      aria-label="Keypad"
      className="grid grid-cols-4 gap-2"
    >
      {buildKeys(props).map((key) => (
        <Key
          key={key.id}
          label={key.label}
          ariaLabel={key.ariaLabel}
          variant={key.variant}
          keyShortcut={key.keyShortcut}
          disabled={props.busy && key.ignoresBusy !== true}
          onPress={key.press}
        />
      ))}
    </div>
  );
}
