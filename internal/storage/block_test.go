package storage

import (
	"os"
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

	fileID := uint32(10)
	offset := int64(100)
	size := uint32(50)
	meta := block.toMeta(fileID, offset, size)

	if meta == nil {
		t.Fatal("ToMeta returned nil")
	}

	if meta.FileID != fileID {
		t.Errorf("Expected FileID %d, got %d", fileID, meta.FileID)
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

func TestSegment_WriteRead(t *testing.T) {
	tmpFile := "test_segment.seg"
	defer os.Remove(tmpFile)

	seg, err := NewSegment(tmpFile, 1)
	if err != nil {
		t.Fatalf("Failed to create segment: %v", err)
	}
	defer seg.Close()

	block := &Block{
		SensorID: 1,
		Points: []Point{
			{Time: 1000, Value: 1.1},
			{Time: 2000, Value: 2.2},
		},
	}

	// Test Write
	meta, err := seg.WriteBlock(block)
	if err != nil {
		t.Fatalf("WriteBlock failed: %v", err)
	}

	if meta.Offset != 0 {
		t.Errorf("Expected offset 0, got %d", meta.Offset)
	}

	// Test Read
	readBlock, err := seg.ReadBlock(meta)
	if err != nil {
		t.Fatalf("ReadBlock failed: %v", err)
	}

	if !reflect.DeepEqual(block, readBlock) {
		t.Errorf("Read block does not match original")
	}
}