type Props = {
  icon?: string;
  className?: string;
};

// Renders a package's icon, or nothing when none is provided. Icons are always
// supplied by the catalog; there is no text or color fallback.
export function PackageIcon({ icon, className }: Props) {
  if (!icon) {
    return null;
  }

  return (
    <img
      src={icon}
      alt=""
      className={`shrink-0 rounded-md object-cover${className ? ` ${className}` : ""}`}
    />
  );
}
