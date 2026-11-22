export function appURL(path) {
    return new URL(path.replace(/^\//, ''), document.baseURI).href
}

export function appCookiePath() {
    return new URL(document.baseURI).pathname.replace(/\/$/, '') || '/'
}
