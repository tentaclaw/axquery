// Package axquery provides a jQuery/goquery-style query API for macOS Accessibility elements.
//
// It builds on the ax package (Level 0) to provide CSS-like selectors, chainable
// Selection operations, and a goja-powered JavaScript runtime for automation scripts.
//
// Core concepts:
//   - Selection: a collection of AX elements with chainable methods
//   - Selector: CSS-like syntax for matching AX elements (e.g., "AXButton[title='OK']")
//   - Matcher: compiled selector for reusable matching
package axquery
