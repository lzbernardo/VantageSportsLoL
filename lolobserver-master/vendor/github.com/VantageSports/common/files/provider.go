package files

// FileManager is a common layer on top of a filesystem for the most basic
// file/object functions. It provides methods for moving files/objects WITHIN
// a single 'system'.
type FileManager interface {
	// Moves the FILE/OBJECT from the source path to the dest path.
	Rename(src, dst string, opts ...FileOption) error

	// Read reads and returns the bytes of src.
	Read(src string) ([]byte, error)

	// Copies the FILE/OBJECT from the source path to the dest path. Note that
	// you cannot copy items across storage providers.
	Copy(src, dst string, opts ...FileOption) error

	// Removes the FILE/OBJECT at the specified path.
	Remove(path string) error

	// If path is a prefix, returns the (full-path) sources therin, otherwise
	// just returns path.
	List(path string) ([]string, error)

	// Returns true if the FILE/OBJECT exists (not necessarily directories).
	Exists(path string) (bool, error)
}

// Remote move files/objects in and out 'systems'.
type Remote interface {
	// DownloadTo returns an io.ReadCloser for the FILE/OBJECT referred to by
	// the specified path.
	DownloadTo(src, localDst string) error

	// UploadTo sends the body defined by the specified io.ReadSeeker to the
	// FILE/OBJECT referred to by the specified path.
	UploadTo(localSrc string, dst string, opts ...FileOption) error

	// AllowPublicRead makes the (remote) path a publically readable object,
	// and returns a url string to reach it.
	AllowPublicRead(dst string) (string, error)

	// Returns a url to the object described by the key in the bucket. Note:
	// this doesn't actually make the object public, it just tells you the URL
	// at which it WOULD be accessible if it were public. Use AllowPublicRead
	// to modify permissions.
	URLFor(src string) (string, error)
}
