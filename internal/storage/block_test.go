package storage

import (
	"reflect"
	"testing"
)

func TestBlock_EncodeDecode(t *testing.T) {
	points := []Point{
		{Time: 1000, Value: 1.1},
		{Time: 2000, Value: 2.2},
		{Time: 3000, Value: 3.3},
	}
	block := &Block{
		SensorID: 1,
		Points:   points,
	}

	// Test Encode
	data, err := block.Encode()
	if err != nil {
		t.Fatalf("Encode failed: %v", err)
	}

	if len(data) == 0 {
		t.Fatal("Encoded data is empty")
	}

	// Test Decode
	decodedBlock, err := DecodeBlock(data)
	if err != nil {
		t.Fatalf("Decode failed: %v", err)
	}

	if !reflect.DeepEqual(block, decodedBlock) {
		t.Errorf("Decoded block does not match original. Got %+v, want %+v", decodedBlock, block)
	}
}

func TestBlock_ToMeta(t *testing.T) {
	points := []Point{
		{Time: 1000, Value: 1.1},
		{Time: 2000, Value: 2.2},
		{Time: 3000, Value: 3.3},
	}
	block := &Block{
		SensorID: 1,
		Points:   points,
	}

	offset := int64(100)
	size := uint32(50)
	meta := block.ToMeta(offset, size)

	if meta == nil {
		t.Fatal("ToMeta returned nil")
	}

	if meta.MinTime != 1000 {
		t.Errorf("Expected MinTime 1000, got %d", meta.MinTime)
	}
	if meta.MaxTime != 3000 {
		t.Errorf("Expected MaxTime 3000, got %d", meta.MaxTime)
	}
	if meta.Offset != offset {
		t.Errorf("Expected Offset %d, got %d", offset, meta.Offset)
	}
	if meta.Size != size {
		t.Errorf("Expected Size %d, got %d", size, meta.Size)
	}
	if meta.Count != 3 {
		t.Errorf("Expected Count 3, got %d", meta.Count)
	}
}

func TestBlock_ToMeta_Empty(t *testing.T) {
	block := &Block{
		SensorID: 1,
		Points:   []Point{},
	}

	meta := block.ToMeta(0, 0)
	if meta != nil {
		t.Error("Expected nil meta for empty block")
	}
}
