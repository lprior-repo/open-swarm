// Copyright (c) 2025 Open Swarm Contributors
//
// This software is released under the MIT License.
// See LICENSE file in the repository for details.

package temporal

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/log"
)

// fileSafeGetLogger returns the activity logger
func fileSafeGetLogger(ctx context.Context) log.Logger {
	return activity.GetLogger(ctx)
}

// fileSafeRecordHeartbeat records a heartbeat if in activity context
func fileSafeRecordHeartbeat(ctx context.Context, details ...interface{}) {
	// Only record heartbeat if we're in an activity context
	defer func() {
		recover()
	}()
	activity.RecordHeartbeat(ctx, details...)
}

// FileActivities provides thin wrappers around file system operations
// These activities are designed to be:
// - Idempotent where possible
// - Minimal business logic
// - Proper error handling
// - Structured results
type FileActivities struct{}

// FileReadInput specifies parameters for reading a file
type FileReadInput struct {
	Path     string // Path to the file to read
	MaxBytes int64  // Maximum bytes to read (0 = no limit)
}

// FileReadOutput contains the result of a file read operation
type FileReadOutput struct {
	Content   string // File contents
	Size      int64  // File size in bytes
	Truncated bool   // Whether content was truncated due to MaxBytes
}

// FileRead reads the contents of a file
// Idempotent: Always returns the current file contents
func (fa *FileActivities) FileRead(ctx context.Context, input FileReadInput) (*FileReadOutput, error) {
	logger := fileSafeGetLogger(ctx)
	logger.Info("Reading file", "path", input.Path)

	fileSafeRecordHeartbeat(ctx, "executing")

	// Get file info
	info, err := os.Stat(input.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to stat file: %w", err)
	}

	if info.IsDir() {
		return nil, fmt.Errorf("path is a directory, not a file: %s", input.Path)
	}

	// Open file
	file, err := os.Open(input.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	result := &FileReadOutput{
		Size:      info.Size(),
		Truncated: false,
	}

	// Read file contents (with optional limit)
	var content []byte
	if input.MaxBytes > 0 && info.Size() > input.MaxBytes {
		content = make([]byte, input.MaxBytes)
		_, err = io.ReadFull(file, content)
		result.Truncated = true
	} else {
		content, err = io.ReadAll(file)
	}

	if err != nil && err != io.EOF {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	result.Content = string(content)

	logger.Info("File read successfully", "size", result.Size, "truncated", result.Truncated)
	return result, nil
}

// FileWriteInput specifies parameters for writing a file
type FileWriteInput struct {
	Path      string // Path to the file to write
	Content   string // Content to write
	Mode      uint32 // File mode (permissions), default 0644
	CreateDir bool   // Whether to create parent directories
	Append    bool   // Whether to append instead of overwrite
}

// FileWriteOutput contains the result of a file write operation
type FileWriteOutput struct {
	Path         string // Path to the written file
	BytesWritten int64  // Number of bytes written
	Created      bool   // Whether the file was newly created
}

// FileWrite writes content to a file
// Idempotent when Append=false: Multiple writes with same content produce same result
func (fa *FileActivities) FileWrite(ctx context.Context, input FileWriteInput) (*FileWriteOutput, error) {
	logger := fileSafeGetLogger(ctx)
	logger.Info("Writing file", "path", input.Path, "append", input.Append)

	fileSafeRecordHeartbeat(ctx, "executing")

	// Create parent directories if requested
	if input.CreateDir {
		dir := filepath.Dir(input.Path)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create directory: %w", err)
		}
	}

	// Check if file exists
	_, err := os.Stat(input.Path)
	created := os.IsNotExist(err)

	// Set default mode if not specified
	mode := fs.FileMode(input.Mode)
	if mode == 0 {
		mode = 0644
	}

	// Open file with appropriate flags
	flags := os.O_WRONLY | os.O_CREATE
	if input.Append {
		flags |= os.O_APPEND
	} else {
		flags |= os.O_TRUNC
	}

	file, err := os.OpenFile(input.Path, flags, mode)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Write content
	n, err := file.WriteString(input.Content)
	if err != nil {
		return nil, fmt.Errorf("failed to write file: %w", err)
	}

	result := &FileWriteOutput{
		Path:         input.Path,
		BytesWritten: int64(n),
		Created:      created,
	}

	logger.Info("File written successfully", "bytes", result.BytesWritten, "created", result.Created)
	return result, nil
}

// FileDeleteInput specifies parameters for deleting a file or directory
type FileDeleteInput struct {
	Path      string // Path to delete
	Recursive bool   // Whether to recursively delete directories
}

// FileDelete deletes a file or directory
// Idempotent: If file doesn't exist, returns success
func (fa *FileActivities) FileDelete(ctx context.Context, input FileDeleteInput) error {
	logger := fileSafeGetLogger(ctx)
	logger.Info("Deleting file", "path", input.Path, "recursive", input.Recursive)

	fileSafeRecordHeartbeat(ctx, "executing")

	// Check if exists
	info, err := os.Stat(input.Path)
	if os.IsNotExist(err) {
		logger.Info("File already deleted (does not exist)")
		return nil // Idempotent
	}
	if err != nil {
		return fmt.Errorf("failed to stat file: %w", err)
	}

	// Delete
	if info.IsDir() {
		if !input.Recursive {
			return fmt.Errorf("path is a directory; use Recursive=true to delete directories")
		}
		if err := os.RemoveAll(input.Path); err != nil {
			return fmt.Errorf("failed to remove directory: %w", err)
		}
	} else {
		if err := os.Remove(input.Path); err != nil {
			return fmt.Errorf("failed to remove file: %w", err)
		}
	}

	logger.Info("File deleted successfully")
	return nil
}

// FileListInput specifies parameters for listing files
type FileListInput struct {
	Path      string // Directory path to list
	Pattern   string // Optional glob pattern (e.g., "*.go")
	Recursive bool   // Whether to recursively list subdirectories
}

// FileInfo represents information about a file
type FileInfo struct {
	Path  string // Relative path from listing root
	Name  string // File name
	Size  int64  // File size in bytes
	IsDir bool   // Whether it's a directory
	Mode  uint32 // File mode (permissions)
}

// FileListOutput contains the result of a file listing
type FileListOutput struct {
	Files []FileInfo // List of files/directories
	Count int        // Total count
}

// FileList lists files in a directory
func (fa *FileActivities) FileList(ctx context.Context, input FileListInput) (*FileListOutput, error) {
	logger := fileSafeGetLogger(ctx)
	logger.Info("Listing files", "path", input.Path, "pattern", input.Pattern, "recursive", input.Recursive)

	fileSafeRecordHeartbeat(ctx, "executing")

	result := &FileListOutput{
		Files: []FileInfo{},
	}

	// Verify path exists and is a directory
	info, err := os.Stat(input.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to stat path: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("path is not a directory: %s", input.Path)
	}

	// List files
	if input.Recursive {
		err = filepath.WalkDir(input.Path, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}

			// Skip root directory itself
			if path == input.Path {
				return nil
			}

			// Apply pattern filter if specified
			if input.Pattern != "" {
				matched, err := filepath.Match(input.Pattern, d.Name())
				if err != nil {
					return fmt.Errorf("invalid pattern: %w", err)
				}
				if !matched && !d.IsDir() {
					return nil
				}
			}

			info, err := d.Info()
			if err != nil {
				return err
			}

			relPath, _ := filepath.Rel(input.Path, path)
			result.Files = append(result.Files, FileInfo{
				Path:  relPath,
				Name:  d.Name(),
				Size:  info.Size(),
				IsDir: d.IsDir(),
				Mode:  uint32(info.Mode()),
			})

			return nil
		})
	} else {
		entries, err := os.ReadDir(input.Path)
		if err != nil {
			return nil, fmt.Errorf("failed to read directory: %w", err)
		}

		for _, entry := range entries {
			// Apply pattern filter if specified
			if input.Pattern != "" {
				matched, err := filepath.Match(input.Pattern, entry.Name())
				if err != nil {
					return nil, fmt.Errorf("invalid pattern: %w", err)
				}
				if !matched {
					continue
				}
			}

			info, err := entry.Info()
			if err != nil {
				continue
			}

			result.Files = append(result.Files, FileInfo{
				Path:  entry.Name(),
				Name:  entry.Name(),
				Size:  info.Size(),
				IsDir: entry.IsDir(),
				Mode:  uint32(info.Mode()),
			})
		}
	}

	if err != nil {
		return nil, fmt.Errorf("failed to walk directory: %w", err)
	}

	result.Count = len(result.Files)

	logger.Info("Files listed successfully", "count", result.Count)
	return result, nil
}

// FileCopyInput specifies parameters for copying a file
type FileCopyInput struct {
	Source      string // Source file path
	Destination string // Destination file path
	CreateDir   bool   // Whether to create parent directories
	Overwrite   bool   // Whether to overwrite existing destination
}

// FileCopyOutput contains the result of a file copy operation
type FileCopyOutput struct {
	Source      string // Source path
	Destination string // Destination path
	BytesCopied int64  // Number of bytes copied
	Overwritten bool   // Whether destination was overwritten
}

// FileCopy copies a file from source to destination
// Idempotent when Overwrite=true: Multiple copies produce same result
func (fa *FileActivities) FileCopy(ctx context.Context, input FileCopyInput) (*FileCopyOutput, error) {
	logger := fileSafeGetLogger(ctx)
	logger.Info("Copying file", "source", input.Source, "destination", input.Destination)

	fileSafeRecordHeartbeat(ctx, "executing")

	// Check source exists
	sourceInfo, err := os.Stat(input.Source)
	if err != nil {
		return nil, fmt.Errorf("failed to stat source: %w", err)
	}
	if sourceInfo.IsDir() {
		return nil, fmt.Errorf("source is a directory; use recursive copy for directories")
	}

	// Check if destination exists
	_, err = os.Stat(input.Destination)
	destExists := !os.IsNotExist(err)
	if destExists && !input.Overwrite {
		return nil, fmt.Errorf("destination exists and Overwrite=false")
	}

	// Create parent directory if requested
	if input.CreateDir {
		dir := filepath.Dir(input.Destination)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create directory: %w", err)
		}
	}

	// Open source
	sourceFile, err := os.Open(input.Source)
	if err != nil {
		return nil, fmt.Errorf("failed to open source: %w", err)
	}
	defer sourceFile.Close()

	// Create destination
	destFile, err := os.OpenFile(input.Destination, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, sourceInfo.Mode())
	if err != nil {
		return nil, fmt.Errorf("failed to create destination: %w", err)
	}
	defer destFile.Close()

	// Copy contents
	bytesCopied, err := io.Copy(destFile, sourceFile)
	if err != nil {
		return nil, fmt.Errorf("failed to copy file: %w", err)
	}

	result := &FileCopyOutput{
		Source:      input.Source,
		Destination: input.Destination,
		BytesCopied: bytesCopied,
		Overwritten: destExists,
	}

	logger.Info("File copied successfully", "bytes", result.BytesCopied)
	return result, nil
}

// FileMoveInput specifies parameters for moving a file
type FileMoveInput struct {
	Source      string // Source file path
	Destination string // Destination file path
	CreateDir   bool   // Whether to create parent directories
	Overwrite   bool   // Whether to overwrite existing destination
}

// FileMove moves or renames a file
// Idempotent: If source doesn't exist and destination exists, assumes already moved
func (fa *FileActivities) FileMove(ctx context.Context, input FileMoveInput) error {
	logger := fileSafeGetLogger(ctx)
	logger.Info("Moving file", "source", input.Source, "destination", input.Destination)

	fileSafeRecordHeartbeat(ctx, "executing")

	// Check source exists
	_, err := os.Stat(input.Source)
	if os.IsNotExist(err) {
		// Check if destination exists (may have already been moved)
		if _, destErr := os.Stat(input.Destination); destErr == nil {
			logger.Info("Source doesn't exist but destination does (already moved)")
			return nil // Idempotent
		}
		return fmt.Errorf("source does not exist: %s", input.Source)
	}
	if err != nil {
		return fmt.Errorf("failed to stat source: %w", err)
	}

	// Check if destination exists
	_, err = os.Stat(input.Destination)
	destExists := !os.IsNotExist(err)
	if destExists && !input.Overwrite {
		return fmt.Errorf("destination exists and Overwrite=false")
	}

	// Create parent directory if requested
	if input.CreateDir {
		dir := filepath.Dir(input.Destination)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory: %w", err)
		}
	}

	// Remove destination if it exists and we're overwriting
	if destExists && input.Overwrite {
		if err := os.Remove(input.Destination); err != nil {
			return fmt.Errorf("failed to remove existing destination: %w", err)
		}
	}

	// Move/rename
	if err := os.Rename(input.Source, input.Destination); err != nil {
		// If rename fails (e.g., across filesystems), fallback to copy+delete
		if strings.Contains(err.Error(), "cross-device") || strings.Contains(err.Error(), "invalid cross-device link") {
			copyInput := FileCopyInput{
				Source:      input.Source,
				Destination: input.Destination,
				CreateDir:   false, // Already created above if needed
				Overwrite:   true,  // Already handled above
			}
			if _, err := fa.FileCopy(ctx, copyInput); err != nil {
				return fmt.Errorf("failed to copy file: %w", err)
			}
			if err := os.Remove(input.Source); err != nil {
				return fmt.Errorf("failed to remove source after copy: %w", err)
			}
		} else {
			return fmt.Errorf("failed to move file: %w", err)
		}
	}

	logger.Info("File moved successfully")
	return nil
}

// FileExistsInput specifies parameters for checking file existence
type FileExistsInput struct {
	Path string // Path to check
}

// FileExistsOutput contains the result of an existence check
type FileExistsOutput struct {
	Exists bool  // Whether the path exists
	IsDir  bool  // Whether it's a directory (false if doesn't exist)
	Size   int64 // File size (0 if doesn't exist or is directory)
}

// FileExists checks if a file or directory exists
func (fa *FileActivities) FileExists(ctx context.Context, input FileExistsInput) (*FileExistsOutput, error) {
	logger := fileSafeGetLogger(ctx)
	logger.Info("Checking file existence", "path", input.Path)

	result := &FileExistsOutput{
		Exists: false,
		IsDir:  false,
		Size:   0,
	}

	info, err := os.Stat(input.Path)
	if os.IsNotExist(err) {
		logger.Info("File does not exist")
		return result, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to stat file: %w", err)
	}

	result.Exists = true
	result.IsDir = info.IsDir()
	if !result.IsDir {
		result.Size = info.Size()
	}

	logger.Info("File existence checked", "exists", result.Exists, "isDir", result.IsDir)
	return result, nil
}
