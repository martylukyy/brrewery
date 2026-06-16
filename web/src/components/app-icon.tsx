type Props = {
  icon?: string;
  className?: string;
};

// Renders an app's icon, or nothing when none is provided. Icons are always
// supplied by the catalog; there is no text or color fallback.
export function AppIcon({ icon, className }: Props) {
  if (!icon) {
    return null;
  }

  return (
    <img
      src={icon}
      alt=""
      className={`shrink-0 object-cover${className ? ` ${className}` : ""}`}
    />
  );
}
