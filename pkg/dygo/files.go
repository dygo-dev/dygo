package dygo

import (
	"context"
	"io"
)

// FileTarget identifies the Record and field that owns an attachment.
type FileTarget struct {
	App      string
	Entity   string
	RecordID int64
	Field    string
}

// FileUpload describes one uploaded file. Target may be empty when the caller
// intends to attach the returned File in a separate operation.
type FileUpload struct {
	Filename    string
	ContentType string
	Size        int64
	Private     bool
	Body        io.Reader
	Target      FileTarget
}

// File is the persisted file descriptor returned by the file SDK.
type File struct {
	ID          int64
	Name        string
	Filename    string
	StorageKey  string
	Checksum    string
	ContentType string
	Size        int64
	Private     bool
	Actor       string
	Retired     bool
	Target      FileTarget
}

// FileData provides authenticated file storage to app code.
type FileData interface {
	Upload(context.Context, FileUpload) (File, error)
	Attach(context.Context, int64, FileTarget) (File, error)
	Open(context.Context, int64) (File, io.ReadCloser, error)
	Remove(context.Context, int64) error
	AsActor(Actor) FileData
}
