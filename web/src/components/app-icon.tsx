import { APP_ICONS } from "@/lib/app-icons";

type Props = {
  // Catalog app id; resolves to a bundled SVG via APP_ICONS.
  appId?: string;
  className?: string;
};

// Renders an app's icon, or nothing when the id has no bundled icon. Icons are
// bundled SVG assets resolved by id (see lib/app-icons); there is no text or
// color fallback.
export function AppIcon({ appId, className }: Props) {
  const src = appId ? APP_ICONS[appId] : undefined;
  if (!src) {
    return null;
  }

  return (
    // object-contain (not -cover) fits the whole logo in its square box without
    // cropping — app icons have varying aspect ratios (e.g. autobrr is wider
    // than tall) and must never be clipped.
    <img
      src={src}
      alt=""
      className={`shrink-0 object-contain${className ? ` ${className}` : ""}`}
    />
  );
}
