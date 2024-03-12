package network

import (
	"testing"
)

func TestCreateNetwork(t *testing.T) {
	Init()
	CreateNetwork("bridge", "192.168.20.1/24", "testbridge")
}

func TestListNetwork(t *testing.T) {
	ListNetwork()
}

func TestDeleteNetwork(t *testing.T) {
	Init()
	DeleteNetwork("testbridge")
}
