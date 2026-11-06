//go:build js

package core

// canQuit is false in the browser: quitting there would only leave a black page.
const canQuit = false
