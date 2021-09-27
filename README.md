# FSNAP – File System Snapshots

Go code (golang) packages that offer concise reading and writing of snapshots of file contents or directory contents.

## Packages

The following packages are included:

### File Snapshots: `fsnap/filesnap`

Read or write a list of file contents.

#### Example

```go
myFiles := filesnap.Files{
	"emptyFile":     []byte(""),
	"intro.txt":     []byte("hello world!"),
	"some/sub/file": []byte("foobar"),
}
myFiles.Write("/target/dir")
```

#### Documentation

- See [`fsnap/filesnap` on pkg.go.dev](https://pkg.go.dev/github.com/echocrow/fsnap/filesnap).

### Directory Snapshot: `fsnap/dirsnap`

Read or write a nested tree of directory entries.

#### Example

```go
type fsd = dirsnap.Dirs

wantEntries := fsd{
	"emptyFile": nil,
	"intro.txt": nil,
	"some":      fsd{"sub": fsd{"file": nil}},
}

gotEntries, err := dirsnap.Read("/target/dir", 5)
require.NoError(t, err)

assert.Equal(t, wantEntries, gotEntries)
```

#### Documentation

- See [`fsnap/dirsnap` on pkg.go.dev](https://pkg.go.dev/github.com/echocrow/fsnap/dirsnap).
