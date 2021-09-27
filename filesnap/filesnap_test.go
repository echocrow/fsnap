package filesnap_test

import (
	"path"
	"testing"

	"github.com/echocrow/fsnap/filesnap"
	"github.com/echocrow/osa"
	tos "github.com/echocrow/osa/testos"
	"github.com/echocrow/osa/vos"
	"github.com/stretchr/testify/assert"
)

type fsf = filesnap.Files

func testScanFiles(
	t *testing.T,
	os osa.I,
	scan func(name string, n int) (fsf, error),
) {
	t.Run("ScanFiles", func(t *testing.T) {
		tmpDir := tos.RequireTempDir(t, os)
		tos.RequireMkdir(t, os, tos.Join(tmpDir, "emptyDir"))
		tos.RequireMkdirAll(t, os, tos.Join(tmpDir, "some", "sub", "dir"))
		tos.RequireEmptyWrite(t, os, tos.Join(tmpDir, "emptyFile"))
		tos.RequireWrite(t, os, tos.Join(tmpDir, "some", "sub", "file"), "foobar")
		tos.RequireWrite(t, os, tos.Join(tmpDir, "some", "nested.txt"), "File Contents")

		tests := []struct {
			name     string
			path     string
			maxDepth int
			want     fsf
		}{
			{
				"Full",
				tmpDir, -1,
				fsf{
					"emptyFile":       []byte(""),
					"some/nested.txt": []byte("File Contents"),
					"some/sub/file":   []byte("foobar"),
				},
			},
			{
				"Nested",
				tos.Join(tmpDir, "some"), -1,
				fsf{
					"nested.txt": []byte("File Contents"),
					"sub/file":   []byte("foobar"),
				},
			},
			{
				"Empty",
				tos.Join(tmpDir, "emptyDir"), -1,
				fsf{},
			},
			{
				"EmptyDepth",
				tmpDir, 0,
				fsf{
					"emptyFile": []byte(""),
				},
			},
			{
				"MaxDepth",
				tos.Join(tmpDir), 1,
				fsf{
					"emptyFile":       []byte(""),
					"some/nested.txt": []byte("File Contents"),
				},
			},
		}
		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				got, err := scan(tc.path, tc.maxDepth)
				assert.Equal(t, tc.want, got)
				assert.NoError(t, err)
			})
		}
	})

	t.Run("ScanFilesErrNotExists", func(t *testing.T) {
		tmpDir := tos.RequireTempDir(t, os)
		missingDir := tos.Join(tmpDir, "missing")

		_, err := scan(missingDir, -1)
		assert.Error(t, err)
		assert.True(t, os.IsNotExist(err), "want not-exists error")
	})
}

func TestScanFiles(t *testing.T) {
	os, reset := vos.Patch()
	defer reset()
	testScanFiles(t, os, filesnap.Read)
}

func TestScanFSFiles(t *testing.T) {
	os, reset := vos.Patch()
	defer reset()
	scan := func(name string, n int) (fsf, error) {
		return filesnap.ReadFS(os, name, n)
	}
	testScanFiles(t, os, scan)
}

func TestWriteFiles(t *testing.T) {
	os, reset := vos.Patch()
	defer reset()

	t.Run("Empty", func(t *testing.T) {
		tmpDir := tos.RequireTempDir(t, os)
		tos.RequireIsEmpty(t, os, tmpDir)

		tr := fsf{}
		err := tr.Write(tmpDir)
		assert.NoError(t, err)
		tos.AssertIsEmpty(t, os, tmpDir)
	})

	t.Run("Main", func(t *testing.T) {
		tmpDir := tos.RequireTempDir(t, os)
		f := fsf{
			"myFile":      []byte("some contents"),
			"emptyFile":   []byte{},
			"nested/file": []byte("foobar"),
		}

		err := f.Write(tmpDir)
		assert.NoError(t, err)

		tos.AssertExists(t, os, tos.Join(tmpDir, "myFile"))
		tos.AssertExists(t, os, tos.Join(tmpDir, "emptyFile"))
		tos.AssertExistsIsDir(t, os, tos.Join(tmpDir, "nested"), true)
		tos.AssertExists(t, os, tos.Join(tmpDir, "nested", "file"))

		t.Run("Again", func(t *testing.T) {
			err := f.Write(tmpDir)
			assert.NoError(t, err)
		})
	})

	t.Run("NilWrite", func(t *testing.T) {
		tmpDir := tos.RequireTempDir(t, os)

		f := fsf{"myFile": nil}
		err := f.Write(tmpDir)
		assert.NoError(t, err)
		tos.AssertFileData(t, os, tos.Join(tmpDir, "myFile"), "")
	})

	t.Run("Overwrite", func(t *testing.T) {
		tmpDir := tos.RequireTempDir(t, os)

		path := tos.Join(tmpDir, "myFile")
		tos.RequireWrite(t, os, path, "old contents")
		newData := "new contents"

		f := fsf{"myFile": []byte(newData)}
		err := f.Write(tmpDir)
		assert.NoError(t, err)
		tos.AssertFileData(t, os, path, newData)
	})

	t.Run("DirExists", func(t *testing.T) {
		tmpDir := tos.RequireTempDir(t, os)

		subPath := tos.Join("nested", "file")
		subDir := path.Dir(subPath)
		dir := tos.Join(tmpDir, subDir)
		path := tos.Join(tmpDir, subPath)
		tos.RequireMkdir(t, os, dir)

		wantData := "some data"
		f := fsf{subPath: []byte(wantData)}
		err := f.Write(tmpDir)
		assert.NoError(t, err)
		tos.AssertExistsIsDir(t, os, dir, true)
		tos.AssertFileData(t, os, path, wantData)
	})

	t.Run("ErrDirCollision", func(t *testing.T) {
		tmpDir := tos.RequireTempDir(t, os)

		name := "myFile"
		path := tos.Join(tmpDir, name)
		tos.RequireMkdir(t, os, path)

		f := fsf{name: []byte("some data")}
		err := f.Write(tmpDir)
		assert.Error(t, err)
		tos.AssertExistsIsDir(t, os, path, true)
	})

	t.Run("ErrNestedDirCollision", func(t *testing.T) {
		tmpDir := tos.RequireTempDir(t, os)

		subPath := tos.Join("nested", "file")
		subDir := path.Dir(subPath)
		dirColl := tos.Join(tmpDir, subDir)
		tos.RequireEmptyWrite(t, os, dirColl)

		f := fsf{
			subPath: []byte("some data"),
		}
		err := f.Write(tmpDir)
		assert.Error(t, err)
		tos.AssertExistsIsDir(t, os, dirColl, false)
	})
}

func TestWriteFilesScanFiles(t *testing.T) {
	os, reset := vos.Patch()
	defer reset()

	tmpDir := tos.RequireTempDir(t, os)

	f := fsf{
		"myFile":      []byte("some contents"),
		"emptyFile":   []byte{},
		"nested/file": []byte("foobar"),
	}

	err := f.Write(tmpDir)
	assert.NoError(t, err)

	gotF, err := filesnap.Read(tmpDir, -1)
	assert.NoError(t, err)
	assert.Equal(t, f, gotF, "expected source and scanned files to match")
}
