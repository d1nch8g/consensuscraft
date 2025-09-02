package fuse

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"syscall"

	"github.com/awnumar/memguard"
	"github.com/hanwen/go-fuse/v2/fs"
	"github.com/hanwen/go-fuse/v2/fuse"
)

type EncryptedFS struct {
	fs.Inode
	key        *memguard.LockedBuffer
	sourcePath string
}

type EncryptedFile struct {
	fs.Inode
	key  *memguard.LockedBuffer
	path string
}

func NewEncryptedFS(sourcePath string, key *memguard.LockedBuffer) *EncryptedFS {
	return &EncryptedFS{
		key:        key,
		sourcePath: sourcePath,
	}
}

func (n *EncryptedFS) OnAdd(ctx context.Context) {
	// Initialize root directory
	if err := os.MkdirAll(n.sourcePath, 0755); err != nil {
		return
	}
}

func (n *EncryptedFS) Lookup(ctx context.Context, name string, out *fuse.EntryOut) (*fs.Inode, syscall.Errno) {
	path := filepath.Join(n.sourcePath, name)
	
	st := syscall.Stat_t{}
	if err := syscall.Lstat(path, &st); err != nil {
		return nil, syscall.ENOENT
	}

	var child fs.InodeEmbedder
	if st.Mode&syscall.S_IFDIR != 0 {
		child = &EncryptedFS{key: n.key, sourcePath: path}
	} else {
		child = &EncryptedFile{key: n.key, path: path}
	}

	out.FromStat(&st)
	return n.NewInode(ctx, child, fs.StableAttr{Mode: st.Mode}), 0
}

func (n *EncryptedFS) Readdir(ctx context.Context) (fs.DirStream, syscall.Errno) {
	entries, err := os.ReadDir(n.sourcePath)
	if err != nil {
		return nil, syscall.EIO
	}

	var result []fuse.DirEntry
	for _, entry := range entries {
		var mode uint32
		if entry.IsDir() {
			mode = syscall.S_IFDIR
		} else {
			mode = syscall.S_IFREG
		}
		result = append(result, fuse.DirEntry{
			Name: entry.Name(),
			Mode: mode,
		})
	}

	return fs.NewListDirStream(result), 0
}

func (f *EncryptedFile) Open(ctx context.Context, flags uint32) (fh fs.FileHandle, fuseFlags uint32, errno syscall.Errno) {
	return &EncryptedFileHandle{key: f.key, path: f.path}, 0, 0
}

type EncryptedFileHandle struct {
	key  *memguard.LockedBuffer
	path string
}

func (fh *EncryptedFileHandle) Read(ctx context.Context, dest []byte, off int64) (fuse.ReadResult, syscall.Errno) {
	// Read encrypted file
	file, err := os.Open(fh.path)
	if err != nil {
		return nil, syscall.EIO
	}
	defer file.Close()

	// Read the nonce (first 12 bytes)
	nonce := make([]byte, 12)
	if _, err := file.Read(nonce); err != nil {
		return nil, syscall.EIO
	}

	// Read encrypted data
	encryptedData, err := io.ReadAll(file)
	if err != nil {
		return nil, syscall.EIO
	}

	// Decrypt
	decryptedData, err := fh.decrypt(encryptedData, nonce)
	if err != nil {
		return nil, syscall.EIO
	}

	// Return requested portion
	if off >= int64(len(decryptedData)) {
		return fuse.ReadResultData([]byte{}), 0
	}

	end := int64(len(decryptedData))
	if off+int64(len(dest)) < end {
		end = off + int64(len(dest))
	}

	return fuse.ReadResultData(decryptedData[off:end]), 0
}

func (fh *EncryptedFileHandle) Write(ctx context.Context, data []byte, off int64) (uint32, syscall.Errno) {
	// Encrypt data
	encryptedData, nonce, err := fh.encrypt(data)
	if err != nil {
		return 0, syscall.EIO
	}

	// Write to file (nonce + encrypted data)
	file, err := os.Create(fh.path)
	if err != nil {
		return 0, syscall.EIO
	}
	defer file.Close()

	if _, err := file.Write(nonce); err != nil {
		return 0, syscall.EIO
	}

	written, err := file.Write(encryptedData)
	if err != nil {
		return 0, syscall.EIO
	}

	return uint32(written), 0
}

func (fh *EncryptedFileHandle) encrypt(data []byte) ([]byte, []byte, error) {
	block, err := aes.NewCipher(fh.key.Bytes())
	if err != nil {
		return nil, nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, nil, err
	}

	encrypted := gcm.Seal(nil, nonce, data, nil)
	return encrypted, nonce, nil
}

func (fh *EncryptedFileHandle) decrypt(data []byte, nonce []byte) ([]byte, error) {
	block, err := aes.NewCipher(fh.key.Bytes())
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	return gcm.Open(nil, nonce, data, nil)
}

func Mount(mountPoint, sourcePath string, key *memguard.LockedBuffer) (*fuse.Server, error) {
	root := NewEncryptedFS(sourcePath, key)

	server, err := fs.Mount(mountPoint, root, &fs.Options{
		MountOptions: fuse.MountOptions{
			Debug: false,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to mount FUSE filesystem: %w", err)
	}

	return server, nil
}
