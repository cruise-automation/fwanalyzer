/*
Copyright 2019-present, Cruise LLC

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package fsparser

type FsParser interface {
	// get directory listing. only returns files in the given directory and
	// does not recurse into subdirectories.
	GetDirInfo(dirpath string) ([]FileInfo, error)
	// get file/dir info
	GetFileInfo(dirpath string) (FileInfo, error)
	// copy (extract) file out of the FS into dest dir
	CopyFile(filepath string, dstDir string) bool
	// get imagename
	ImageName() string
	// determine if FS type is supported
	Supported() bool
}

type FileInfo struct {
	Size         int64    `json:"size"`
	Mode         uint64   `json:"mode"`
	Uid          int      `json:"uid"`
	Gid          int      `json:"gid"`
	SELinuxLabel string   `json:"se_linux_label,omitempty"`
	Capabilities []string `json:"capabilities,omitempty"`
	Name         string   `json:"name"`
	LinkTarget   string   `json:"link_target,omitempty"`
}

const (
	SELinuxNoLabel string = "-"
)

const (
	S_IFMT   = 0170000 // bit mask for the file type bit fields
	S_IFSOCK = 0140000 // socket
	S_IFLNK  = 0120000 // symbolic link
	S_IFREG  = 0100000 // regular file
	S_IFBLK  = 0060000 // block device
	S_IFDIR  = 0040000 // directory
	S_IFCHR  = 0020000 // character device
	S_IFIFO  = 0010000 // FIFO
	S_ISUID  = 0004000 // set-user-ID bit
	S_ISGID  = 0002000 // set-group-ID bit (see below)
	S_ISVTX  = 0001000 // sticky bit (see below)
	S_IRWXU  = 00700   // mask for file owner permissions
	S_IRUSR  = 00400   // owner has read permission
	S_IWUSR  = 00200   // owner has write permission
	S_IXUSR  = 00100   // owner has execute permission
	S_IRWXG  = 00070   // mask for group permissions
	S_IRGRP  = 00040   // group has read permission
	S_IWGRP  = 00020   // group has write permission
	S_IXGRP  = 00010   // group has execute permission
	S_IRWXO  = 00007   // mask for permissions for others (not in group)
	S_IROTH  = 00004   // others have read permission
	S_IWOTH  = 00002   // others have write permission
	S_IXOTH  = 00001   // others have execute permission
)

func (fi *FileInfo) IsSUid() bool {
	return (fi.Mode & S_ISUID) != 0
}

func (fi *FileInfo) IsSGid() bool {
	return (fi.Mode & S_ISGID) != 0
}

func (fi *FileInfo) IsWorldWrite() bool {
	return (fi.Mode & S_IWOTH) != 0
}

func (fi *FileInfo) IsFile() bool {
	return (fi.Mode & S_IFMT) == S_IFREG
}

func (fi *FileInfo) IsDir() bool {
	return (fi.Mode & S_IFMT) == S_IFDIR
}

func (fi *FileInfo) IsLink() bool {
	return (fi.Mode & S_IFMT) == S_IFLNK
}
