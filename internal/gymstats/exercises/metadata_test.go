package exercises

import "testing"

func TestMetadataToStrings(t *testing.T) {
	got, err := metadataToStrings([]byte(`{"testing":"false","plates":1.25,"ok":true,"skip":null}`), 7)
	if err != nil {
		t.Fatal(err)
	}
	if got["testing"] != "false" || got["plates"] != "1.25" || got["ok"] != "true" {
		t.Fatalf("metadata = %#v", got)
	}
	if _, ok := got["skip"]; ok {
		t.Fatalf("nil metadata value was kept: %#v", got)
	}
}
