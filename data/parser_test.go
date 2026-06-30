package data_test

import (
	"testing"

	"venera/data"
)

func TestParseJSONToPairs(t *testing.T) {
	jsonData := []byte(`{"ip":"192.168.1.1", "nested": {"port": 80, "protocol":"tcp:ip"}}`)
	ts := int64(1625000000000)

	pairs, err := data.ParseJSONToPairs(jsonData, ts)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(pairs) != 3 {
		t.Errorf("Expected 3 pairs, got %d", len(pairs))
	}
	
	// В зависимости от порядка обхода ключи могут быть разными, но проверим наличие конкретного
	foundNested := false
	for _, p := range pairs {
		key, val, pTs, err := data.ImprovedParseEntry(p)
		if err != nil {
			t.Errorf("ImprovedParseEntry error on %s: %v", p, err)
		}
		if pTs != ts {
			t.Errorf("Expected ts %d, got %d", ts, pTs)
		}
		
		if key == "nested.protocol" {
			foundNested = true
			if val != "tcp:ip" {
				t.Errorf("Expected value 'tcp:ip', got '%s'", val)
			}
		}
	}
	
	if !foundNested {
		t.Errorf("Did not find key 'nested.protocol'")
	}
}

func TestImprovedParseEntry(t *testing.T) {
	entry := "nested.key:value:with:colons:1625000000000"
	
	key, val, ts, err := data.ImprovedParseEntry(entry)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	
	if key != "nested.key" {
		t.Errorf("Expected key 'nested.key', got '%s'", key)
	}
	
	if val != "value:with:colons" {
		t.Errorf("Expected val 'value:with:colons', got '%s'", val)
	}
	
	if ts != 1625000000000 {
		t.Errorf("Expected ts 1625000000000, got %d", ts)
	}
}
