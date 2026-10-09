// Package gfx draws the game and reports input, on whichever platform the
// binary was built for.
//
// The game asks for very little: solid rectangles, blitted sprites, and one
// inverting rectangle. Every platform therefore implements the same six
// drawing operations on Canvas, and the frame loop hands each tick's mouse
// and keyboard state to the game as an Input value rather than keeping it
// where anything can read it.
//
// The game draws in logical pixels on a board of a fixed size, and the canvas
// maps them onto however many pixels the window or page has. A sprite is
// either a raster, scaled with its pixels kept whole, or something drawn
// afresh at whatever scale the canvas settles on so that it stays sharp at
// any size: a vector drawing, or text set from a font.
//
// Two implementations sit behind build tags. The browser build draws to a 2D
// canvas through syscall/js, leaves the vector drawings to the browser, and
// has no third-party dependency at all; every other build wraps Ebitengine
// and rasterises the drawings with oksvg. Choosing between them at compile
// time rather than at run time is what keeps the browser build small,
// because Go's linker drops a package nothing reachable from main can call.
package gfx
