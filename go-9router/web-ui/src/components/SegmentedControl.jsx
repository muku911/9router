import { cn } from "../lib/cn";

const sizes = {
  sm: "h-7 text-xs",
  md: "h-9 text-sm",
  lg: "h-11 text-base",
};

export default function SegmentedControl({
  options = [],
  value,
  onChange,
  size = "md",
  className,
}) {
  return (
    <div
      className={cn(
        "inline-flex items-center rounded-lg bg-surface-2/80 border border-border-subtle p-0.5",
        className
      )}
    >
      {options.map((option) => {
        const isActive = option.value === value;
        return (
          <button
            key={option.value}
            onClick={() => onChange(option.value)}
            className={cn(
              "inline-flex items-center justify-center gap-1.5 px-3 rounded-md font-medium transition-all duration-200 cursor-pointer",
              sizes[size],
              isActive
                ? "bg-surface text-text-main shadow-sm"
                : "text-text-muted hover:text-text-main"
            )}
          >
            {option.icon && (
              <span className="material-symbols-outlined text-[16px]">{option.icon}</span>
            )}
            {option.label}
          </button>
        );
      })}
    </div>
  );
}
