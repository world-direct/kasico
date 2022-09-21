package controllers

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHashStringMap_Empty(t *testing.T) {

	m1 := make(map[string]string)
	h1 := HashStringMap(m1)
	assert.Greater(t, len(h1), 0)

	m2 := make(map[string]string)
	h2 := HashStringMap(m2)
	assert.Equal(t, h1, h2)
}

func TestHashStringMap_One(t *testing.T) {
	m1 := make(map[string]string)
	m2 := make(map[string]string)

	m1["data"] = "v1"
	m2["data"] = "v1"
	h1 := HashStringMap(m1)
	h2 := HashStringMap(m2)
	assert.Equal(t, h1, h2)
}

func TestHashStringMap_Two(t *testing.T) {
	m1 := make(map[string]string)
	m2 := make(map[string]string)

	m1["data"] = "v1"
	m2["data"] = "v1"
	m1["data2"] = "v2"
	m2["data2"] = "v2"
	h1 := HashStringMap(m1)
	h2 := HashStringMap(m2)
	assert.Equal(t, h1, h2)
}

func TestHashStringMap_DifferentKeys(t *testing.T) {
	m1 := make(map[string]string)
	m2 := make(map[string]string)

	m1["data"] = "v1"
	m2["data"] = "v1"
	m1["data2"] = "v2"
	m2["data3"] = "v2"
	h1 := HashStringMap(m1)
	h2 := HashStringMap(m2)
	assert.NotEqual(t, h1, h2)
}

func TestHashStringMap_DifferentValues(t *testing.T) {
	m1 := make(map[string]string)
	m2 := make(map[string]string)

	m1["data"] = "v1"
	m2["data"] = "v1"
	m1["data2"] = "v2"
	m2["data2"] = "v3"
	h1 := HashStringMap(m1)
	h2 := HashStringMap(m2)
	assert.NotEqual(t, h1, h2)
}
