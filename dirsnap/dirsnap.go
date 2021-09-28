// Package dirsnap simplifies reading and writing snapshots of directories and
// their contents.
//
// Here is an example of how to use this package during testing:
//
//	import (
//		"testing"
//
//		"github.com/echocrow/fsnap/dirsnap"
//		"github.com/stretchr/testify/assert"
//		"github.com/stretchr/testify/require"
//	)
//
//	type fsd = dirsnap.Dirs
//
//	func TestMyFilesPurge(t *testing.T) {
//		dir := t.TempDir()
//
//		// Source dirs snapshot.
//		require.NoError(t, fsd{
//			"file": nil,
//			"dir": fsd{
//				"subfile": nil,
//				"subdir":  fsd{},
//			},
//		}.Write(dir))
//
//		// Expected dirs snapshot.
//		want := fsd{
//			"dir": fsd{
//				"subdir": fsd{},
//			},
//		}
//
//		err := MyFilesPurge(dir)
//		assert.NoError(t, err)
//
//		// Actual dirs snapshot.
//		got, err := dirsnap.Read(dir, -1)
//		require.NoError(t, err)
//		assert.Equal(t, want, got)
//	}
//
package dirsnap

import (
	"io"
	"io/fs"
	"path"

	os "github.com/echocrow/osa"
)

// Dirs represents a nested snapshot of directory contents.
//
// Keys represent the name of a file or folder. Files are represented by a
// nil value and subfolders by a nested Dirs instance.
type Dirs map[string]Dirs

// Read scans a directory and returs a Dirs tree of its files and folders.
//
// If n < 0, Read will scan all subdirectories.
//
// If n >= 0, Read will descend at most n directory levels below directory dir.
func Read(dir string, n int) (Dirs, error) {
	osa := os.Current()
	return ReadFS(osa, dir, n)
}

// ReadFS scans a fsys directory dir and returs a Dirs tree of its files and
// folders.
//
// See Read().
func ReadFS(fsys fs.FS, dir string, n int) (Dirs, error) {
	t := Dirs{}

	entries, err := fs.ReadDir(fsys, dir)
	if err != nil && err != io.EOF {
		return nil, err
	}
	for _, e := range entries {
		en := e.Name()
		if !e.IsDir() {
			t[en] = nil
		} else if n == 0 {
			t[en] = Dirs{}
		} else {
			var err error
			t[en], err = ReadFS(fsys, path.Join(dir, en), n-1)
			if err != nil {
				return t, err
			}
		}
	}

	return t, nil
}

// Write writes Dirs d into directory dir, creating new files and folders
// accordingly.
//
// Collisions with already existing files or folders will not result in errors
// as long as they are of the same type (directory or file respectively).
func (d Dirs) Write(dir string) error {
	for n, st := range d {
		name := path.Join(dir, n)
		if st == nil {
			// Handle file.
			if err := writeEmptyFile(name); !d.isWriteErrOk(err, name, false) {
				return err
			}
		} else {
			// Handle dir.
			if err := os.Mkdir(name, 0700); !d.isWriteErrOk(err, name, true) {
				return err
			}
			if err := st.Write(name); err != nil {
				return err
			}
		}
	}
	return nil
}

func (t Dirs) isWriteErrOk(err error, name string, wantDir bool) bool {
	if !os.IsExist(err) {
		return err == nil
	}
	fi, err := os.Stat(name)
	gotDir := fi != nil && fi.IsDir()
	return err == nil && gotDir == wantDir
}

func writeEmptyFile(name string) error {
	return os.WriteFile(name, []byte{}, 0600)
}
