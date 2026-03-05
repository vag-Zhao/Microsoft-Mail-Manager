package security

import "testing"

func TestSanitizeHTML_PreservesOriginalHTML(t *testing.T) {
	input := `<html><body style="margin:0"><script>window.test=1</script><div onclick="alert(1)" style="position:absolute;left:10px">Hello</div><a href="javascript:alert(1)">link</a></body></html>`

	got := SanitizeHTML(input)
	if got != input {
		t.Fatalf("SanitizeHTML should preserve original HTML\nwant: %s\n got: %s", input, got)
	}
}
