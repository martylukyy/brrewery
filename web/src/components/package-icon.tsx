type Props = {
  icon?: string;
};

// Renders a package's icon, or nothing when none is provided. Icons are always
// supplied by the catalog; there is no text or color fallback.
export function PackageIcon({ icon }: Props) {
  if (!icon) {
    return null;
  }

  return (
    <img
      src={icon}
      alt=""
      className="h-5 w-5 shrink-0 rounded-md object-cover"
    />
  );
}
