package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseChangeLabels(t *testing.T) {
	// standard 4-label config
	labels, ok := parseChangeLabels("change::emergency,change::major,change::normal,change::standard")
	assert.True(t, ok)
	assert.Equal(t, []string{"change::emergency", "change::major", "change::normal", "change::standard"}, labels)

	// 3 labels
	labels, ok = parseChangeLabels("change::emergency,change::normal,change::standard")
	assert.True(t, ok)
	assert.Equal(t, []string{"change::emergency", "change::normal", "change::standard"}, labels)

	// 5 labels
	labels, ok = parseChangeLabels("change::emergency,change::major,change::normal,change::minor,change::standard")
	assert.True(t, ok)
	assert.Len(t, labels, 5)

	// minimum: exactly 2 labels
	labels, ok = parseChangeLabels("change::important,change::standard")
	assert.True(t, ok)
	assert.Equal(t, []string{"change::important", "change::standard"}, labels)

	// too few: only 1 label
	_, ok = parseChangeLabels("change::emergency")
	assert.False(t, ok)

	// empty
	_, ok = parseChangeLabels("")
	assert.False(t, ok)

	// duplicates
	_, ok = parseChangeLabels("change::emergency,change::major,change::normal,change::normal")
	assert.False(t, ok)

	// labels without :: separator
	_, ok = parseChangeLabels("emergency,major,normal,standard")
	assert.False(t, ok)
}

func TestParseChangeLabelDefault(t *testing.T) {
	priority := []string{"change::emergency", "change::major", "change::normal", "change::standard"}

	dl, ok := parseChangeLabelDefault("change::standard", priority)
	assert.True(t, ok)
	assert.Equal(t, "change::standard", dl)

	// empty string is valid (no fallback)
	dl, ok = parseChangeLabelDefault("", priority)
	assert.True(t, ok)
	assert.Equal(t, "", dl)

	// label not in priority list
	_, ok = parseChangeLabelDefault("change::unknown", priority)
	assert.False(t, ok)
}

func TestParseChangeLabelMap(t *testing.T) {
	priority := []string{"change::emergency", "change::major", "change::normal", "change::standard"}

	mapping, ok := parseChangeLabelMap("feat=change::normal,chore=change::standard,ops=change::standard", priority)
	assert.True(t, ok)
	assert.Equal(t, map[string]string{
		"feat":  "change::normal",
		"chore": "change::standard",
		"ops":   "change::standard",
	}, mapping)

	mapping, ok = parseChangeLabelMap("", priority)
	assert.True(t, ok)
	assert.Empty(t, mapping)

	_, ok = parseChangeLabelMap("feat=change::normal,feat=change::standard", priority)
	assert.False(t, ok)

	_, ok = parseChangeLabelMap("feat=change::critical", priority)
	assert.False(t, ok)

	_, ok = parseChangeLabelMap("feat-change::normal", priority)
	assert.False(t, ok)
}

