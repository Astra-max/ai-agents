/**
 * Handles is active path
 */
const isActivePath = () => {
    const path = window.location.pathname

    document.querySelectorAll(".nav-a").forEach( link => {
        const href = link.getAttribute('href')
        const isHome = href === "/" && path === "/"
        const isActive = href !== "/" && path.startsWith(href)

        if (isHome) link.classList.add("active")
})
}

isActivePath()