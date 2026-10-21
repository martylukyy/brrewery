import * as React from "react"

// Radix portals a dialog's overlay and content into <body> as two separate
// nodes, so whether the overlay paints below the content (both z-50) depends
// only on their order among <body>'s children. Browser extensions that keep
// their own elements last in <body> (seen with a password manager) move newly
// added nodes in front of them last-first, which reverses that order: the blur
// overlay then covers the content and swallows its clicks. Portaling both into
// one container of our own keeps them in order wherever the container is moved,
// while Radix still mounts and unmounts each one (exit animations included).
export function usePortalContainer(slot: string) {
  const [container] = React.useState(() => {
    const element = document.createElement("div")
    element.dataset.slot = slot
    return element
  })

  React.useLayoutEffect(() => {
    document.body.appendChild(container)
    return () => container.remove()
  }, [container])

  return container
}
