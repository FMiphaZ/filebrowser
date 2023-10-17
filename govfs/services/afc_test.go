package services_test

import (
	"fmt"
	"io"
	"testing"
	"time"

	"git.woa.com/frosthuang/ga-filebrowser/govfs/services"
)

func TestAfcService(t *testing.T) {
	afc, err := services.NewAfcService("127.0.0.1:5001")
	if err != nil {
		t.Fatal(err)
	}
	defer afc.Close()

	list, err := afc.ReadDir("./")
	if err != nil {
		t.Fatal(err)
	}

	fmt.Println("command line ls:")
	for _, l := range list {
		fmt.Println(l)
	}
}

func TestAfcServiceStat(t *testing.T) {
	afc, err := services.NewAfcService("127.0.0.1:5001")
	if err != nil {
		t.Fatal(err)
	}
	defer afc.Close()

	stat, err := afc.Stat("/Users/frosthuang/Work/code/ga-filebrowser/testdir/")
	if err != nil {
		t.Fatal(err)
	}

	fmt.Printf("%+v\n", stat)
}

func TestAfcServiceRm(t *testing.T) {
	afc, err := services.NewAfcService("127.0.0.1:5001")
	if err != nil {
		t.Fatal(err)
	}
	defer afc.Close()

	err = afc.RemovePath("/Users/frosthuang/Work/code/ga-filebrowser/testdir")
	if err != nil {
		t.Fatal(err)
	}

	list, err := afc.ReadDir("/Users/frosthuang/Work/code/ga-filebrowser/")
	if err != nil {
		t.Fatal(err)
	}

	fmt.Println("command line ls:")
	for _, l := range list {
		fmt.Println(l)
	}
}

func TestAfcServiceRename(t *testing.T) {
	afc, err := services.NewAfcService("127.0.0.1:5001")
	if err != nil {
		t.Fatal(err)
	}
	defer afc.Close()

	err = afc.RenamePath("/Users/frosthuang/Work/code/ga-filebrowser/testdir/testfile1", "testfile2")
	if err != nil {
		t.Fatal(err)
	}

	list, err := afc.ReadDir("/Users/frosthuang/Work/code/ga-filebrowser/testdir")
	if err != nil {
		t.Fatal(err)
	}

	fmt.Println("command line ls:")
	for _, l := range list {
		fmt.Println(l)
	}
}

func TestAfcServiceMakedir(t *testing.T) {
	afc, err := services.NewAfcService("127.0.0.1:5001")
	if err != nil {
		t.Fatal(err)
	}
	defer afc.Close()

	err = afc.MakeDir("/Users/frosthuang/Work/code/ga-filebrowser/testdir")
	if err != nil {
		t.Fatal(err)
	}

	list, err := afc.ReadDir("/Users/frosthuang/Work/code/ga-filebrowser/")
	if err != nil {
		t.Fatal(err)
	}

	fmt.Println("command line ls:")
	for _, l := range list {
		fmt.Println(l)
	}
}

// Openfile and Closefile
func TestAfcServiceOpenfile(t *testing.T) {
	afc, err := services.NewAfcService("127.0.0.1:5001")
	if err != nil {
		t.Fatal(err)
	}
	defer afc.Close()

	fd, err := afc.OpenFile("/Users/frosthuang/Work/code/ga-filebrowser/testdir/testfile", services.Afc_Mode_WR)
	if err != nil {
		t.Fatal(err)
	}

	err = afc.CloseFile(fd)
	if err != nil {
		t.Fatal(err)
	}

	fmt.Printf("fd: %d\n", fd)
}

func TestAfcServiceRead(t *testing.T) {
	afc, err := services.NewAfcService("127.0.0.1:5001")
	if err != nil {
		t.Fatal(err)
	}
	defer afc.Close()

	fd, err := afc.OpenFile("/Users/frosthuang/Work/code/ga-filebrowser/testdir/testfile", services.Afc_Mode_RDONLY)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Printf("fd: %d\n", fd)

	p := make([]byte, 32)
	n, err := afc.ReadFile(fd, p)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Printf("out bytes num=%d\n", n)
	for _, l := range p {
		fmt.Printf("%c", rune(l))
	}

	err = afc.CloseFile(fd)
	if err != nil {
		t.Fatal(err)
	}

	fmt.Printf("fd: %d\n", fd)
}

func TestAfcServiceWrite(t *testing.T) {
	afc, err := services.NewAfcService("127.0.0.1:5001")
	if err != nil {
		t.Fatal(err)
	}
	defer afc.Close()

	fd, err := afc.OpenFile("/Users/frosthuang/Work/code/ga-filebrowser/testdir/testfile", services.Afc_Mode_WR)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Printf("fd: %d\n", fd)

	p := []byte{'1', '2', '3'}
	_, err = afc.WriteFile(fd, p)
	if err != nil {
		t.Fatal(err)
	}

	err = afc.CloseFile(fd)
	if err != nil {
		t.Fatal(err)
	}

	fmt.Printf("fd: %d\n", fd)
}

// Seekfile and Tellfile
func TestAfcServiceSeek(t *testing.T) {
	afc, err := services.NewAfcService("127.0.0.1:5001")
	if err != nil {
		t.Fatal(err)
	}
	defer afc.Close()

	fd, err := afc.OpenFile("/Users/frosthuang/Work/code/ga-filebrowser/testdir/testfile", services.Afc_Mode_WR)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Printf("fd: %d\n", fd)

	off_set, err := afc.SeekFile(fd, 1, io.SeekStart)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Printf("off_set: %d\n", off_set)

	err = afc.CloseFile(fd)
	if err != nil {
		t.Fatal(err)
	}

	fmt.Printf("fd: %d\n", fd)
}

func TestAfcServiceTruncate(t *testing.T) {
	afc, err := services.NewAfcService("127.0.0.1:5001")
	if err != nil {
		t.Fatal(err)
	}
	defer afc.Close()

	fd, err := afc.OpenFile("/Users/frosthuang/Work/code/ga-filebrowser/testdir/testfile", services.Afc_Mode_RW)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Printf("fd: %d\n", fd)

	stat, _ := afc.Stat("/Users/frosthuang/Work/code/ga-filebrowser/testdir/testfile")
	fmt.Printf("old_size=%d\n", stat.Size())
	err = afc.TruncateFile(fd, 5)
	if err != nil {
		t.Fatal(err)
	}
	stat, _ = afc.Stat("/Users/frosthuang/Work/code/ga-filebrowser/testdir/testfile")
	fmt.Printf("new_size=%d\n", stat.Size())

	_ = afc.Truncate("/Users/frosthuang/Work/code/ga-filebrowser/testdir/testfile", 3)
	stat, _ = afc.Stat("/Users/frosthuang/Work/code/ga-filebrowser/testdir/testfile")
	fmt.Printf("new_new_size=%d\n", stat.Size())

	err = afc.CloseFile(fd)
	if err != nil {
		t.Fatal(err)
	}

	fmt.Printf("fd: %d\n", fd)
}

func TestAfcServiceSetFileTime(t *testing.T) {
	afc, err := services.NewAfcService("127.0.0.1:5001")
	if err != nil {
		t.Fatal(err)
	}
	defer afc.Close()

	new_time := time.Now()

	stat, _ := afc.Stat("/Users/frosthuang/Work/code/ga-filebrowser/testdir/testfile")
	fmt.Printf("old_time=%s\n", stat.ModTime().Format("2006-01-02 15:04:05"))
	err = afc.SetFileTime("/Users/frosthuang/Work/code/ga-filebrowser/testdir/testfile", new_time)
	if err != nil {
		t.Fatal(err)
	}
	stat, _ = afc.Stat("/Users/frosthuang/Work/code/ga-filebrowser/testdir/testfile")
	fmt.Printf("new_time=%s\n", stat.ModTime().Format("2006-01-02 15:04:05"))
}
