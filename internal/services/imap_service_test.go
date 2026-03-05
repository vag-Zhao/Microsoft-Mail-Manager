package services

import "testing"

func TestSanitizeHTML_PreservesOriginalHTML(t *testing.T) {
	input := `<div><script>window.test=1</script><img src="x" onerror="alert(1)" style="width:100px;height:50px"></div>`

	got := sanitizeHTML(input)
	if got != input {
		t.Fatalf("sanitizeHTML should preserve original HTML\nwant: %s\n got: %s", input, got)
	}
}
