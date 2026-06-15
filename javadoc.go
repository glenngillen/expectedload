package expectedload

// stripJavadoc removes Javadoc/KDoc block-comment framing. The grammar is
// identical to JSDoc (`/** ... * ... */`), so it reuses the JSDoc stripper.
//
//	/**
//	 * @expected-load
//	 *   monthlyCalls = 5_000_000
//	 */
func stripJavadoc(line string) string {
	return stripJSDoc(line)
}
