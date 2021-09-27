// Package filesnap simplifies reading and writing snapshots of files and their
// contents.
//
// Here is an example of how to use this package during testing:
//
//	import (
//		"testing"
//
//		"github.com/echocrow/fsnap/filesnap"
//		"github.com/stretchr/testify/assert"
//		"github.com/stretchr/testify/require"
//	)
//
//	type fsf = filesnap.Files
//
//	func TestMyFilesConcat(t *testing.T) {
//		dir := t.TempDir()
//
//		// Source files snapshot.
//		require.NoError(t, fsf{
//			"a_0.txt": []byte("prefix"),
//			"a_1.txt": []byte("suffix"),
//		}.Write(dir))
//
//		// Expected files snapshot.
//		want := fsf{
//			"a.txt": []byte("prefix\nsuffix"),
//		}
//
//		err := MyFilesConcat(dir)
//		assert.NoError(t, err)
//
//		// Actual files snapshot.
//		got, err := filesnap.Read(dir, -1)
//		require.NoError(t, err)
//		assert.Equal(t, want, got)
//	}
//
package filesnap

import (
	"io"
	"io/fs"
	"path"

	os "github.com/echocrow/osa"
)

// Files represents a snapshot of files in a directory and its subdirectories.
//
// Keys represent the subpath of the files and their values are the contents of
// the respective file.
type Files map[string][]byte

// Read scans a directory and returs its Files.
//
// If n < 0, Read will scan all subdirectories.
//
// If n >= 0, Read will descend at most n directory levels below the given
// directory.
func Read(dir string, n int) (Files, error) {
	osa := os.Current()
	return ReadFS(osa, dir, n)
}

// ReadFS scans a fsys directory and returs its Files.
//
// See Read().
func ReadFS(fsys fs.FS, dir string, n int) (Files, error) {
	f := Files{}
	err := f.readFS(fsys, dir, "", n)
	return f, err
}

func (f Files) readFS(fsys fs.FS, rootDir, subDir string, n int) error {
	dir := path.Join(rootDir, subDir)
	entries, err := fs.ReadDir(fsys, dir)
	if err != nil && err != io.EOF {
		return err
	}
	for _, e := range entries {
		p := path.Join(dir, e.Name())
		sp := path.Join(subDir, e.Name())
		if !e.IsDir() {
			var err error
			f[sp], err = fs.ReadFile(fsys, p)
			if err != nil && err != io.EOF {
				return err
			}
		} else if n != 0 {
			if err := f.readFS(fsys, rootDir, sp, n-1); err != nil {
				return err
			}
		}
	}

	return nil
}

// Write writes Files f into directory dir, creating new files and folders
// accordingly.
//
// Already existing colliding file will be overwritten.
func (f Files) Write(dir string) error {
	for n, data := range f {
		p := path.Join(dir, n)
		dir := path.Dir(p)
		if err := os.MkdirAll(dir, 0700); err != nil {
			return err
		}
		if err := os.WriteFile(p, data, 0600); err != nil {
			return err
		}
	}
	return nil
}
