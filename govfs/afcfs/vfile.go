package afcfs

import (
	"os"
	"path"
	"syscall"

	"git.woa.com/CloudTesting/UDT/ioskit/ioskit/services"
)

type VFile struct {
	absPath string
	names   []string
}

func (f *VFile) Close() (err error) {
	return nil
}

func (f *VFile) Read(p []byte) (n int, err error) {
	return 0, syscall.EPFNOSUPPORT
}

func (f *VFile) ReadAt(p []byte, off int64) (n int, err error) {
	return 0, syscall.EPFNOSUPPORT
}

func (f *VFile) Seek(offset int64, whence int) (int64, error) {
	return 0, syscall.EPFNOSUPPORT
}

func (f *VFile) Write(p []byte) (n int, err error) {
	return 0, syscall.EPFNOSUPPORT
}

func (f *VFile) WriteAt(p []byte, off int64) (n int, err error) {
	return 0, syscall.EPFNOSUPPORT
}

func (f *VFile) Name() string {
	return f.absPath
}

func (f *VFile) Readdir(count int) (fi []os.FileInfo, err error) {
	if count > 0 {
		return nil, syscall.EPFNOSUPPORT
	}

	for _, name := range f.names {
		fi = append(fi, services.NewDirStatInfo(name))
	}
	return
}

func (f *VFile) Readdirnames(count int) (names []string, err error) {
	if count > 0 {
		return nil, syscall.EPFNOSUPPORT
	}
	return f.names, err
}

func (f *VFile) Stat() (os.FileInfo, error) {
	return services.NewDirStatInfo(path.Base(f.absPath)), nil
}

func (f *VFile) Sync() error {
	return nil
}

func (f *VFile) Truncate(size int64) error {
	return syscall.EPFNOSUPPORT
}

func (f *VFile) WriteString(s string) (ret int, err error) {
	return -1, syscall.EPFNOSUPPORT
}
