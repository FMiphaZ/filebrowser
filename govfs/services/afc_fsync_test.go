package services_test

import (
	"fmt"
	"testing"

	"git.woa.com/frosthuang/ga-filebrowser/govfs/services"
)

func TestAfcSyncTree(t *testing.T) {
	afc, err := services.NewFsync("127.0.0.1:5001")
	if err != nil {
		t.Fatal(err)
	}
	defer afc.Close()

	err = afc.TreeView("/Users/frosthuang/Work/code/ga-filebrowser/govfs", "", false)
	if err != nil {
		t.Fatal(err)
	}
}

func TestAfcSyncPush(t *testing.T) {
	afc, err := services.NewFsync("127.0.0.1:5001")
	if err != nil {
		t.Fatal(err)
	}
	defer afc.Close()

	err = afc.PushWithHandler("/Users/frosthuang/Downloads/goland-2023.2.1-aarch64.dmg", "/Users/frosthuang/Work/code/ga-filebrowser", func(size uint64, status string) {
		fmt.Printf("size:%d status:%s\n", size, status)
	})

	if err != nil {
		t.Fatal(err)
	}
}

func TestAfcSyncPull(t *testing.T) {
	afc, err := services.NewFsync("127.0.0.1:5001")
	if err != nil {
		t.Fatal(err)
	}
	defer afc.Close()

	err = afc.Pull("/Users/frosthuang/Downloads/goland-2023.2.1-aarch64.dmg", "/Users/frosthuang/Work/code/ga-filebrowser/largefile")

	if err != nil {
		t.Fatal(err)
	}
}
