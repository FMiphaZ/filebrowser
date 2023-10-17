package services

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"syscall"
	"time"

	"git.woa.com/CloudTesting/UDT/ioskit/ioskit/utils"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/afero"
)

type FsType int

const (
	AfcAnyFs               FsType = 0
	AfcRootFs              FsType = 1
	HouseArrestContainerFs FsType = 2
	HouseArrestDocumentFs  FsType = 3
	CrashReportFs          FsType = 4
)

type Fsync struct {
	*AfcService
	FsType   FsType
	BundleId string //only used for house_arrest
}

func NewFsync(addr string) (*Fsync, error) {
	afc, err := NewAfcService(addr)
	if err != nil {
		return nil, err
	}
	return &Fsync{afc, AfcRootFs, ""}, nil
}

/*
func NewFsyncFromConn() *Fsync {
	return &Fsync{&AfcService{deviceConn: devConn}, AfcAnyFs, ""}
}
*/

func (fs *Fsync) SendFile(b []byte, path string) error {
	fd, err := fs.AfcService.OpenFile(path, Afc_Mode_WRONLY)
	if err != nil {
		return err
	}
	defer fs.CloseFile(fd)
	_, err = fs.AfcService.WriteFile(fd, b)
	if err != nil {
		return err
	}
	return nil
}

// ListFiles returns all files in the given directory, matching the pattern.
// Example: ListFiles(".", "*") returns all files and dirs in the current path the afc connection is in
func (fs *Fsync) ListFiles(cwd string, matchPattern string) ([]string, error) {
	files, err := fs.ReadDir(cwd)
	if err != nil {
		return nil, err
	}

	var filteredFiles []string
	for _, f := range files {
		if f == "" {
			continue
		}
		matches, err := filepath.Match(matchPattern, f)
		if err != nil {
			log.Warn("error while matching pattern", err)
		}
		if matches {
			filteredFiles = append(filteredFiles, f)
		}
	}
	return filteredFiles, nil
}

func (fs *Fsync) TreeView(dpath string, prefix string, treePoint bool) error {
	fileInfo, err := fs.Stat(dpath)
	if err != nil {
		return err
	}

	namePrefix := "`--"
	if !treePoint {
		namePrefix = "|--"
	}
	tPrefix := prefix + namePrefix
	if !fileInfo.IsDir() {
		//return fmt.Errorf("error: %v is not dir", dpath)
		fmt.Printf("%s %s\n", tPrefix, filepath.Base(dpath))
		return nil
	}

	fmt.Printf("%s %s/\n", tPrefix, filepath.Base(dpath))
	fileList, err := fs.ReadDir(dpath)
	if err != nil {
		return err
	}
	for i, v := range fileList {
		tp := false
		if i == len(fileList)-1 {
			tp = true
		}
		rp := prefix + "    "
		if !treePoint {
			rp = prefix + "|   "
		}
		nPath := path.Join(dpath, v)
		err = fs.TreeView(nPath, rp, tp)
		if err != nil {
			return err
		}
	}

	return nil
}

func (fs *Fsync) PullFile(srcPath, dstPath string) error {
	fileInfo, err := fs.AfcService.Stat(srcPath)
	if err != nil {
		return err
	}

	if fileInfo.IsLink() {
		srcPath = fileInfo.stLinktarget
	}
	fd, err := fs.AfcService.OpenFile(srcPath, Afc_Mode_RDONLY)
	if err != nil {
		return err
	}
	defer fs.CloseFile(fd)

	f, err := os.Create(dstPath)
	if err != nil {
		return err
	}
	defer f.Close()

	leftSize := fileInfo.stSize
	maxReadSize := 64 * 1024
	data := make([]byte, 16)
	binary.LittleEndian.PutUint64(data, fd)
	binary.LittleEndian.PutUint64(data[8:], uint64(maxReadSize))
	for leftSize > 0 {
		response, err := fs.request(Afc_operation_file_read, data, nil)
		if err != nil {
			return err
		}
		leftSize -= int64(len(response.Payload))
		f.Write(response.Payload)
	}
	return nil
}

func (fs *Fsync) Pull(srcPath, dstPath string) error {
	fileInfo, err := fs.Stat(srcPath)
	if err != nil {
		return err
	}
	if !fileInfo.IsDir() {
		return fs.PullFile(srcPath, dstPath)
	}
	ret, _ := utils.PathExists(dstPath)
	if !ret {
		err = os.MkdirAll(dstPath, 0755)
		if err != nil {
			return err
		}
	}
	fileList, err := fs.ReadDir(srcPath)
	if err != nil {
		return err
	}
	for _, v := range fileList {
		sp := path.Join(srcPath, v)
		dp := path.Join(dstPath, v)
		err = fs.Pull(sp, dp)
		if err != nil {
			return err
		}
	}
	return nil
}

func (fs *Fsync) Push(srcPath, dstPath string) error {
	return fs.PushWithHandler(srcPath, dstPath, nil)
}

func (fs *Fsync) PushWithHandler(srcPath, dstPath string, handler func(size uint64, status string)) error {
	ret, _ := utils.PathExists(srcPath)
	if !ret {
		return fmt.Errorf("%s: no such file", srcPath)
	}

	f, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer f.Close()

	fileInfo, err := fs.Stat(dstPath)
	if err == nil {
		if fileInfo.IsDir() {
			dstPath = path.Join(dstPath, filepath.Base(srcPath))
		}
	}

	info, err := os.Stat(srcPath)
	if err != nil {
		return err
	}
	fileSize := info.Size()

	fd, err := fs.AfcService.OpenFile(dstPath, Afc_Mode_WR)
	if err != nil {
		return err
	}
	defer fs.CloseFile(fd)

	// NOTE: optimize memory cost
	var maxWriteSize int
	if fileSize < 1024*1024 {
		maxWriteSize = 128 * 1024
	} else {
		maxWriteSize = 1024 * 1024
	}
	chunk := make([]byte, maxWriteSize)
	for {
		n, err := f.Read(chunk)
		if err != nil && err != io.EOF {
			return err
		}
		if n == 0 {
			break
		}
		if handler != nil {
			handler(uint64(n), "Pushing")
		}

		data := make([]byte, 8)
		binary.LittleEndian.PutUint64(data, fd)
		_, err = fs.request(Afc_operation_file_write, data, chunk[0:n])
		if err != nil {
			return err
		}
	}
	return nil
}

func (fs *Fsync) Name() string { return "iosfs" }

func (fs *Fsync) Create(name string) (afero.File, error) {
	fd, err := fs.AfcService.OpenFile(name, Afc_Mode_WR) // O_RDWR | O_CREAT | O_TRUNC
	if err != nil {
		return nil, &os.PathError{Op: "open", Path: name, Err: syscall.ENOENT}
	}
	return &File{pfd: fd, absPath: name, conn: fs.AfcService}, nil
}

func (fs *Fsync) Mkdir(name string, perm os.FileMode) error {
	return fs.MakeDir(name)
}

func (fs *Fsync) MkdirAll(path string, perm os.FileMode) error {
	info, err := fs.AfcService.Stat(path)
	if err != nil {
		return fs.MakeDir(path)
	}

	if info.IsDir() {
		return nil
	}
	return fmt.Errorf("path:%v is not directory", path)
}

func (fs *Fsync) Open(name string) (afero.File, error) {
	return fs.OpenFile(name, os.O_RDONLY, 0)
}

// OpenFile see https://github.com/libimobiledevice/ifuse/blob/master/src/ifuse.c#L177
func (fs *Fsync) OpenFile(name string, flag int, perm os.FileMode) (afero.File, error) {
	info, err := fs.AfcService.Stat(name)
	if err == nil {
		if info.IsDir() {
			return &File{absPath: name, conn: fs.AfcService, isdir: true}, nil
		}
	}

	var afcFlags uint64
	switch flag & 0x03 {
	case os.O_RDONLY:
		afcFlags = Afc_Mode_RDONLY
	case os.O_WRONLY:
		{
			if flag&os.O_TRUNC != 0 {
				afcFlags = Afc_Mode_WRONLY
			} else if flag&os.O_APPEND != 0 {
				afcFlags = Afc_Mode_APPEND
			} else {
				afcFlags = Afc_Mode_RW
			}
		}
	case os.O_RDWR:
		{
			if flag&os.O_TRUNC != 0 {
				afcFlags = Afc_Mode_WR
			} else if flag&os.O_APPEND != 0 {
				afcFlags = Afc_Mode_RDAPPEND
			} else {
				afcFlags = Afc_Mode_RW
			}
		}
	default:
		return nil, fmt.Errorf("invalid flag")
	}

	fd, err := fs.AfcService.OpenFile(name, afcFlags)
	if err != nil {
		return nil, err
	}

	return &File{pfd: fd, absPath: name, conn: fs.AfcService, isdir: false}, nil
}

func (fs *Fsync) Remove(name string) error {
	return fs.RemovePath(name)
}

func (fs *Fsync) RemoveAll(path string) error {
	return fs.RemovePathAndContents(path)
}

func (fs *Fsync) Rename(oldname, newname string) error {
	return fs.RenamePath(oldname, newname)
}

func (fs *Fsync) Stat(name string) (os.FileInfo, error) {
	return fs.AfcService.Stat(name)
}

func (fs *Fsync) Chmod(name string, mode os.FileMode) error {
	return nil
}

func (fs *Fsync) Chown(name string, uid, gid int) error {
	return nil
}

func (fs *Fsync) Chtimes(name string, atime time.Time, mtime time.Time) error {
	return nil
}

func (fs *Fsync) RmTree(path string) error {
	info, err := fs.AfcService.Stat(path)
	if err != nil {
		return err
	}
	if info.IsDir() {
		files, err := fs.AfcService.ReadDir(path)
		if err != nil {
			return err
		}
		for _, f := range files {
			filePath := path + "/" + f
			info, err = fs.AfcService.Stat(filePath)
			if err != nil {
				log.Errorf("stat %v error: %v", f, err)
				continue
			}
			if info.IsDir() {
				err = fs.RmTree(filePath)
				if err != nil {
					return err
				}
			} else {
				err = fs.AfcService.RemovePath(filePath)
				if err != nil {
					return err
				}
			}
		}
		err = fs.AfcService.RemovePath(path)
		if err != nil {
			return err
		}
	} else {
		err = fs.AfcService.RemovePath(path)
		if err != nil {
			return err
		}
	}
	return err
}
