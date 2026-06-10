package file

import "io"

// FileUpload represents a file upload request.
// Callers must not close File themselves — UploadFile closes it after the transfer.
type FileUpload struct {
	Filename    string
	Size        int64
	ContentType string
	File        io.ReadCloser
}
