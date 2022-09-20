package controllers

import (
	"testing"

	"github.com/stretchr/testify/assert"
	kasicov1 "github.com/world-direct/kasico/api/v1"
)

func TestRemoveFirst(t *testing.T) {
	ingresses := []kasicov1.IngressReference{
		{Namespace: "default", Name: "name1"},
		{Namespace: "default", Name: "name2"},
		{Namespace: "default", Name: "name3"},
		{Namespace: "default", Name: "name4"},
	}

	res := removeIngressReferenceFromSlice(ingresses, 0)
	assert.Equal(t, 3, len(res))
	assert.Equal(t, "name2", res[0].Name)
	assert.Equal(t, "name3", res[1].Name)
	assert.Equal(t, "name4", res[2].Name)
}

func TestRemoveLast(t *testing.T) {
	ingresses := []kasicov1.IngressReference{
		{Namespace: "default", Name: "name1"},
		{Namespace: "default", Name: "name2"},
		{Namespace: "default", Name: "name3"},
		{Namespace: "default", Name: "name4"},
	}

	res := removeIngressReferenceFromSlice(ingresses, 3)
	assert.Equal(t, 3, len(res))
	assert.Equal(t, "name1", res[0].Name)
	assert.Equal(t, "name2", res[1].Name)
	assert.Equal(t, "name3", res[2].Name)
}

func TestRemoveMiddle(t *testing.T) {
	ingresses := []kasicov1.IngressReference{
		{Namespace: "default", Name: "name1"},
		{Namespace: "default", Name: "name2"},
		{Namespace: "default", Name: "name3"},
		{Namespace: "default", Name: "name4"},
	}

	res := removeIngressReferenceFromSlice(ingresses, 1)
	assert.Equal(t, 3, len(res))
	assert.Equal(t, "name1", res[0].Name)
	assert.Equal(t, "name3", res[1].Name)
	assert.Equal(t, "name4", res[2].Name)
}
