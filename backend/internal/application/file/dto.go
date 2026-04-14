package file

import "io"

// FileUpload represents a file upload request.
type FileUpload struct {
	Filename    string
	Size        int64
	ContentType string
	File        io.Reader
}
