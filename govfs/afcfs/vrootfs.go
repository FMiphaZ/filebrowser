package afcfs

import (
	"os"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/filebrowser/filebrowser/v2/govfs/services"
	"github.com/spf13/afero"
)

const (
	afcMountPath = ""
	// crashreportsMountPath = "/crashreports"
	// sandboxMountPath      = "/apps"
	// documentsPath         = "/Documents"
	documentsDirName = "Documents"
)

type VirtualRootFs struct {
	afero.Fs
	addr        string
	mountPoints map[string]*services.Fsync
}

func NewVfs(addr string) (*VirtualRootFs, error) {
	afcFs, err := services.NewFsync(addr)
	if err != nil {
		return nil, err
	}

	rootFs := &VirtualRootFs{
		addr:        addr,
		mountPoints: make(map[string]*services.Fsync),
	}
	rootFs.Mount(afcMountPath, afcFs)
	return rootFs, nil
}

func (fs *VirtualRootFs) trimPath(path string, mountPoint string) string {
	trimmedPath := strings.TrimPrefix(path, mountPoint)
	if !strings.HasPrefix(trimmedPath, "/") {
		trimmedPath = "/" + trimmedPath
	}
	return trimmedPath
}

func (fs *VirtualRootFs) findMountPoint(filepath string) (f afero.Fs, p string) {
	nf, _, np := fs.findMountPoint2(filepath)
	return nf, np
}

func (fs *VirtualRootFs) findMountPoint2(filepath string) (f afero.Fs, mountPoint, newPath string) {
	for mp, f := range fs.mountPoints {
		if strings.HasPrefix(filepath, mp) {
			np := strings.TrimPrefix(filepath, mp)
			return f, mp, np
		}
	}
	return nil, "", ""
}

func (fs *VirtualRootFs) Mount(mountPath string, vfs *services.Fsync) {
	fs.mountPoints[mountPath] = vfs
}

func (fs *VirtualRootFs) Unmount(mountPath string) {
	// TODO: need call fs.Unmount
	delete(fs.mountPoints, mountPath)
}

func winPathToUnix(name string) string {
	if runtime.GOOS == "windows" {
		name = strings.ReplaceAll(name, "\\", "/")
	}
	return name
}

func (fs *VirtualRootFs) Create(name string) (afero.File, error) {
	name = winPathToUnix(name)

	mp, newPath := fs.findMountPoint(name)
	if mp == nil {
		return nil, syscall.EPERM
	}
	return mp.Create(newPath)
}

func (fs *VirtualRootFs) Mkdir(name string, perm os.FileMode) error {
	name = winPathToUnix(name)

	mp, newPath := fs.findMountPoint(name)
	if mp == nil {
		return syscall.EPERM
	}

	return mp.Mkdir(newPath, perm)
}

func (fs *VirtualRootFs) MkdirAll(name string, perm os.FileMode) error {
	name = winPathToUnix(name)

	mp, newPath := fs.findMountPoint(name)
	if mp == nil {
		return syscall.EPERM
	}

	return mp.MkdirAll(newPath, perm)
}

func (fs *VirtualRootFs) Open(name string) (afero.File, error) {
	name = winPathToUnix(name)

	return fs.OpenFile(name, os.O_RDONLY, 0)
}

// OpenFile see https://github.com/libimobiledevice/ifuse/blob/master/src/ifuse.c#L177
func (fs *VirtualRootFs) OpenFile(name string, flag int, perm os.FileMode) (afero.File, error) {
	name = winPathToUnix(name)

	f, newPath := fs.findMountPoint(name)
	if f != nil {
		return f.OpenFile(newPath, flag, perm)
	}
	return nil, syscall.ENOENT
}

func (fs *VirtualRootFs) Remove(name string) error {
	name = winPathToUnix(name)

	mp, newPath := fs.findMountPoint(name)
	if mp == nil {
		return syscall.EPERM
	}

	return mp.Remove(newPath)
}

func (fs *VirtualRootFs) RemoveAll(name string) error {
	name = winPathToUnix(name)

	mp, newPath := fs.findMountPoint(name)
	if mp == nil {
		return syscall.EPERM
	}

	return mp.RemoveAll(newPath)
}

func (fs *VirtualRootFs) Rename(oldname, newname string) error {
	oldname = winPathToUnix(oldname)
	newname = winPathToUnix(newname)

	mp, point, oldname2 := fs.findMountPoint2(oldname)
	if mp == nil {
		return syscall.EPERM
	}
	newname2 := fs.trimPath(newname, point)
	return mp.Rename(oldname2, newname2)
}

func (fs *VirtualRootFs) Stat(name string) (os.FileInfo, error) {
	name = winPathToUnix(name)

	mp, newPath := fs.findMountPoint(name)
	if mp == nil {
		return services.NewDirStatInfo(name), nil
	}

	return mp.Stat(newPath)
}

func (fs *VirtualRootFs) Name() string { return "iOSVirtualRootFs" }

func (fs *VirtualRootFs) Chmod(name string, mode os.FileMode) error {
	name = winPathToUnix(name)

	mp, newPath := fs.findMountPoint(name)
	if mp == nil {
		return syscall.EPERM
	}
	return mp.Chmod(newPath, mode)
}

func (fs *VirtualRootFs) Chown(name string, uid, gid int) error {
	name = winPathToUnix(name)

	mp, newPath := fs.findMountPoint(name)
	if mp == nil {
		return syscall.EPERM
	}
	return mp.Chown(newPath, uid, gid)
}

func (fs *VirtualRootFs) Chtimes(name string, atime time.Time, mtime time.Time) error {
	name = winPathToUnix(name)

	mp, newPath := fs.findMountPoint(name)
	if mp == nil {
		return syscall.EPERM
	}

	return mp.Chtimes(newPath, atime, mtime)
}
