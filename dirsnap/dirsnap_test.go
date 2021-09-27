package dirsnap_test

import (
	"testing"

	"github.com/echocrow/fsnap/dirsnap"
	"github.com/echocrow/osa"
	tos "github.com/echocrow/osa/testos"
	"github.com/echocrow/osa/vos"
	"github.com/stretchr/testify/assert"
)

type fsd = dirsnap.Dirs

func testScanTree(
	t *testing.T,
	os osa.I,
	scan func(name string, n int) (fsd, error),
) {
	t.Run("ScanTree", func(t *testing.T) {
		tmpDir := tos.RequireTempDir(t, os)
		tos.RequireMkdir(t, os, tos.Join(tmpDir, "emptyDir"))
		tos.RequireMkdirAll(t, os, tos.Join(tmpDir, "some", "sub", "dir"))
		tos.RequireEmptyWrite(t, os, tos.Join(tmpDir, "some", "nested.txt"))
		tos.RequireEmptyWrite(t, os, tos.Join(tmpDir, "anotherFile"))

		tests := []struct {
			name string
			dir  string
			maxD int
			want fsd
		}{
			{
				"Full",
				tmpDir, -1,
				fsd{
					"emptyDir": fsd{},
					"some": fsd{
						"sub": fsd{
							"dir": fsd{},
						},
						"nested.txt": nil,
					},
					"anotherFile": nil,
				},
			},
			{
				"Nested",
				tos.Join(tmpDir, "some", "sub"), -1,
				fsd{
					"dir": fsd{},
				},
			},
			{
				"Empty",
				tos.Join(tmpDir, "emptyDir"), -1,
				fsd{},
			},
			{
				"EmptyDepth",
				tmpDir, 0,
				fsd{
					"emptyDir":    fsd{},
					"some":        fsd{},
					"anotherFile": nil,
				},
			},
			{
				"MaxDepth",
				tos.Join(tmpDir), 1,
				fsd{
					"emptyDir": fsd{},
					"some": fsd{
						"sub":        fsd{},
						"nested.txt": nil,
					},
					"anotherFile": nil,
				},
			},
		}
		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				got, err := scan(tc.dir, tc.maxD)
				assert.Equal(t, tc.want, got)
				assert.NoError(t, err)
			})
		}
	})

	t.Run("ScanTreeErrNotExists", func(t *testing.T) {
		tmpDir := tos.RequireTempDir(t, os)
		missingDir := tos.Join(tmpDir, "missing")

		_, err := scan(missingDir, -1)
		assert.Error(t, err)
		assert.True(t, os.IsNotExist(err), "want not-exists error")
	})
}

func TestScanTree(t *testing.T) {
	os, reset := vos.Patch()
	defer reset()
	testScanTree(t, os, dirsnap.Read)
}

func TestScanFSTree(t *testing.T) {
	os, reset := vos.Patch()
	defer reset()
	scan := func(name string, n int) (fsd, error) {
		return dirsnap.ReadFS(os, name, n)
	}
	testScanTree(t, os, scan)
}

func TestWriteTree(t *testing.T) {
	os, reset := vos.Patch()
	defer reset()

	t.Run("Empty", func(t *testing.T) {
		tmpDir := tos.RequireTempDir(t, os)
		tos.RequireIsEmpty(t, os, tmpDir)

		tr := fsd{}
		err := tr.Write(tmpDir)
		assert.NoError(t, err)
		tos.AssertIsEmpty(t, os, tmpDir)
	})

	t.Run("Main", func(t *testing.T) {
		tmpDir := tos.RequireTempDir(t, os)
		tr := fsd{
			"emptyDir": fsd{},
			"some": fsd{
				"subDir":     fsd{},
				"nested.txt": nil,
			},
			"anotherFile": nil,
		}

		err := tr.Write(tmpDir)
		assert.NoError(t, err)

		tos.AssertExistsIsDir(t, os, tos.Join(tmpDir, "emptyDir"), true)
		tos.AssertExistsIsDir(t, os, tos.Join(tmpDir, "some"), true)
		tos.AssertExistsIsDir(t, os, tos.Join(tmpDir, "some", "subDir"), true)
		tos.AssertExistsIsDir(t, os, tos.Join(tmpDir, "some", "nested.txt"), false)
		tos.AssertExistsIsDir(t, os, tos.Join(tmpDir, "anotherFile"), false)

		t.Run("Again", func(t *testing.T) {
			err := tr.Write(tmpDir)
			assert.NoError(t, err)
		})
	})

	t.Run("ErrFileCollision", func(t *testing.T) {
		tmpDir := tos.RequireTempDir(t, os)
		n := "collFile"
		tos.RequireEmptyWrite(t, os, tos.Join(tmpDir, n))

		tr := fsd{n: fsd{}}
		err := tr.Write(tmpDir)
		assert.Error(t, err)
	})

	t.Run("ErrDirCollision", func(t *testing.T) {
		tmpDir := tos.RequireTempDir(t, os)
		n := "collDir"
		tos.RequireMkdir(t, os, tos.Join(tmpDir, n))

		tr := fsd{n: nil}
		err := tr.Write(tmpDir)
		assert.Error(t, err)
	})
}

func TestWriteTreeScanTree(t *testing.T) {
	os, reset := vos.Patch()
	defer reset()

	tmpDir := tos.RequireTempDir(t, os)

	tr := fsd{
		"emptyDir": fsd{},
		"some": fsd{
			"sub": fsd{
				"dir": fsd{},
			},
			"nested.txt": nil,
		},
		"anotherFile": nil,
	}

	err := tr.Write(tmpDir)
	assert.NoError(t, err)

	gotTr, err := dirsnap.Read(tmpDir, -1)
	assert.NoError(t, err)
	assert.Equal(t, tr, gotTr, "expected source and scanned tree to match")
}
