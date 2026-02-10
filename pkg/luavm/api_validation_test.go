package luavm

import (
	"testing"
)

func TestConcurrencySafety(t *testing.T) {
	done := make(chan bool, 2)

	go func() {
		L := NewVM(nil)
		defer L.Close()
		L.DoString(`local s = bk.image("alpine:3.19"); bk.export(s)`)
		done <- true
	}()

	go func() {
		L := NewVM(nil)
		defer L.Close()
		L.DoString(`local s = bk.image("ubuntu:24.04"); bk.export(s)`)
		done <- true
	}()

	<-done
	<-done
}
