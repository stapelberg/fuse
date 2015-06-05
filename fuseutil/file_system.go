// Copyright 2015 Google Inc. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package fuseutil

import (
	"flag"
	"io"
	"math/rand"
	"time"

	"github.com/jacobsa/fuse"
	"github.com/jacobsa/fuse/fuseops"
)

var fRandomDelays = flag.Bool(
	"fuseutil.random_delays", false,
	"If set, randomly delay each op received, to help expose concurrency issues.")

// An interface with a method for each op type in the fuseops package. This can
// be used in conjunction with NewFileSystemServer to avoid writing a "dispatch
// loop" that switches on op types, instead receiving typed method calls
// directly.
//
// The FileSystem implementation should not call Op.Respond, instead returning
// the error with which the caller should respond.
//
// See NotImplementedFileSystem for a convenient way to embed default
// implementations for methods you don't care about.
type FileSystem interface {
	Init(*fuseops.InitOp) error
	LookUpInode(*fuseops.LookUpInodeOp) error
	GetInodeAttributes(*fuseops.GetInodeAttributesOp) error
	SetInodeAttributes(*fuseops.SetInodeAttributesOp) error
	ForgetInode(*fuseops.ForgetInodeOp) error
	MkDir(*fuseops.MkDirOp) error
	CreateFile(*fuseops.CreateFileOp) error
	CreateSymlink(*fuseops.CreateSymlinkOp) error
	RmDir(*fuseops.RmDirOp) error
	Unlink(*fuseops.UnlinkOp) error
	OpenDir(*fuseops.OpenDirOp) error
	ReadDir(*fuseops.ReadDirOp) error
	ReleaseDirHandle(*fuseops.ReleaseDirHandleOp) error
	OpenFile(*fuseops.OpenFileOp) error
	ReadFile(*fuseops.ReadFileOp) error
	WriteFile(*fuseops.WriteFileOp) error
	SyncFile(*fuseops.SyncFileOp) error
	FlushFile(*fuseops.FlushFileOp) error
	ReleaseFileHandle(*fuseops.ReleaseFileHandleOp) error
	ReadSymlink(*fuseops.ReadSymlinkOp) error
}

// Create a fuse.Server that handles ops by calling the associated FileSystem
// method.Respond with the resulting error. Unsupported ops are responded to
// directly with ENOSYS.
//
// Each call to a FileSystem method is made on its own goroutine, and is free
// to block.
//
// (It is safe to naively process ops concurrently because the kernel
// guarantees to serialize operations that the user expects to happen in order,
// cf. http://goo.gl/jnkHPO, fuse-devel thread "Fuse guarantees on concurrent
// requests").
func NewFileSystemServer(fs FileSystem) fuse.Server {
	return fileSystemServer{fs}
}

type fileSystemServer struct {
	fs FileSystem
}

func (s fileSystemServer) ServeOps(c *fuse.Connection) {
	for {
		op, err := c.ReadOp()
		if err == io.EOF {
			break
		}

		if err != nil {
			panic(err)
		}

		go s.handleOp(op)
	}
}

func (s fileSystemServer) handleOp(op fuseops.Op) {
	// Delay if requested.
	if *fRandomDelays {
		const delayLimit = 100 * time.Microsecond
		delay := time.Duration(rand.Int63n(int64(delayLimit)))
		time.Sleep(delay)
	}

	// Dispatch to the appropriate method.
	var err error
	switch typed := op.(type) {
	default:
		err = fuse.ENOSYS

	case *fuseops.InitOp:
		err = s.fs.Init(typed)

	case *fuseops.LookUpInodeOp:
		err = s.fs.LookUpInode(typed)

	case *fuseops.GetInodeAttributesOp:
		err = s.fs.GetInodeAttributes(typed)

	case *fuseops.SetInodeAttributesOp:
		err = s.fs.SetInodeAttributes(typed)

	case *fuseops.ForgetInodeOp:
		err = s.fs.ForgetInode(typed)

	case *fuseops.MkDirOp:
		err = s.fs.MkDir(typed)

	case *fuseops.CreateFileOp:
		err = s.fs.CreateFile(typed)

	case *fuseops.CreateSymlinkOp:
		err = s.fs.CreateSymlink(typed)

	case *fuseops.RmDirOp:
		err = s.fs.RmDir(typed)

	case *fuseops.UnlinkOp:
		err = s.fs.Unlink(typed)

	case *fuseops.OpenDirOp:
		err = s.fs.OpenDir(typed)

	case *fuseops.ReadDirOp:
		err = s.fs.ReadDir(typed)

	case *fuseops.ReleaseDirHandleOp:
		err = s.fs.ReleaseDirHandle(typed)

	case *fuseops.OpenFileOp:
		err = s.fs.OpenFile(typed)

	case *fuseops.ReadFileOp:
		err = s.fs.ReadFile(typed)

	case *fuseops.WriteFileOp:
		err = s.fs.WriteFile(typed)

	case *fuseops.SyncFileOp:
		err = s.fs.SyncFile(typed)

	case *fuseops.FlushFileOp:
		err = s.fs.FlushFile(typed)

	case *fuseops.ReleaseFileHandleOp:
		err = s.fs.ReleaseFileHandle(typed)

	case *fuseops.ReadSymlinkOp:
		err = s.fs.ReadSymlink(typed)
	}

	op.Respond(err)
}
