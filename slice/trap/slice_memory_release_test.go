package main

import (
	"testing"
)

func TestGetLastBySlice(t *testing.T) { testGetLast(t, GetLastBySlice) }
func TestGetLastByCopy(t *testing.T)  { testGetLast(t, GetLastByCopy) }
