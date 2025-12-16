// Copyright (c) 2025 Open Swarm Contributors
//
// This software is released under the MIT License.
// See LICENSE file in the repository for details.

package temporal

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFileActivities_FileRead(t *testing.T) {
	fa := &FileActivities{}
	ctx := context.Background()

	t.Run("read existing file", func(t *testing.T) {
		tmpDir := t.TempDir()
		testFile := filepath.Join(tmpDir, "test.txt")
		content := "Hello, World!"
		require.NoError(t, os.WriteFile(testFile, []byte(content), 0644))

		input := FileReadInput{
			Path: testFile,
		}

		output, err := fa.FileRead(ctx, input)
		require.NoError(t, err)
		assert.Equal(t, content, output.Content)
		assert.Equal(t, int64(len(content)), output.Size)
		assert.False(t, output.Truncated)
	})

	t.Run("read with max bytes limit", func(t *testing.T) {
		tmpDir := t.TempDir()
		testFile := filepath.Join(tmpDir, "large.txt")
		content := "This is a longer content that will be truncated"
		require.NoError(t, os.WriteFile(testFile, []byte(content), 0644))

		input := FileReadInput{
			Path:     testFile,
			MaxBytes: 10,
		}

		output, err := fa.FileRead(ctx, input)
		require.NoError(t, err)
		assert.Equal(t, "This is a ", output.Content)
		assert.Equal(t, int64(len(content)), output.Size)
		assert.True(t, output.Truncated)
	})

	t.Run("file not found", func(t *testing.T) {
		input := FileReadInput{
			Path: "/nonexistent/file.txt",
		}

		_, err := fa.FileRead(ctx, input)
		assert.Error(t, err)
	})

	t.Run("path is directory", func(t *testing.T) {
		tmpDir := t.TempDir()

		input := FileReadInput{
			Path: tmpDir,
		}

		_, err := fa.FileRead(ctx, input)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "directory")
	})
}

func TestFileActivities_FileWrite(t *testing.T) {
	fa := &FileActivities{}
	ctx := context.Background()

	t.Run("write new file", func(t *testing.T) {
		tmpDir := t.TempDir()
		testFile := filepath.Join(tmpDir, "new.txt")
		content := "New content"

		input := FileWriteInput{
			Path:    testFile,
			Content: content,
		}

		output, err := fa.FileWrite(ctx, input)
		require.NoError(t, err)
		assert.Equal(t, testFile, output.Path)
		assert.Equal(t, int64(len(content)), output.BytesWritten)
		assert.True(t, output.Created)

		// Verify file was written
		data, err := os.ReadFile(testFile)
		require.NoError(t, err)
		assert.Equal(t, content, string(data))
	})

	t.Run("overwrite existing file", func(t *testing.T) {
		tmpDir := t.TempDir()
		testFile := filepath.Join(tmpDir, "existing.txt")
		require.NoError(t, os.WriteFile(testFile, []byte("old"), 0644))

		input := FileWriteInput{
			Path:    testFile,
			Content: "new content",
		}

		output, err := fa.FileWrite(ctx, input)
		require.NoError(t, err)
		assert.False(t, output.Created)

		// Verify content was replaced
		data, err := os.ReadFile(testFile)
		require.NoError(t, err)
		assert.Equal(t, "new content", string(data))
	})

	t.Run("append to file", func(t *testing.T) {
		tmpDir := t.TempDir()
		testFile := filepath.Join(tmpDir, "append.txt")
		require.NoError(t, os.WriteFile(testFile, []byte("initial"), 0644))

		input := FileWriteInput{
			Path:    testFile,
			Content: " appended",
			Append:  true,
		}

		output, err := fa.FileWrite(ctx, input)
		require.NoError(t, err)
		assert.Equal(t, int64(9), output.BytesWritten)

		// Verify content was appended
		data, err := os.ReadFile(testFile)
		require.NoError(t, err)
		assert.Equal(t, "initial appended", string(data))
	})

	t.Run("create parent directories", func(t *testing.T) {
		tmpDir := t.TempDir()
		testFile := filepath.Join(tmpDir, "nested", "dir", "file.txt")

		input := FileWriteInput{
			Path:      testFile,
			Content:   "content",
			CreateDir: true,
		}

		output, err := fa.FileWrite(ctx, input)
		require.NoError(t, err)
		assert.True(t, output.Created)

		// Verify file and directories were created
		_, err = os.Stat(testFile)
		require.NoError(t, err)
	})

	t.Run("custom file mode", func(t *testing.T) {
		tmpDir := t.TempDir()
		testFile := filepath.Join(tmpDir, "mode.txt")

		input := FileWriteInput{
			Path:    testFile,
			Content: "content",
			Mode:    0600,
		}

		_, err := fa.FileWrite(ctx, input)
		require.NoError(t, err)

		// Verify file mode
		info, err := os.Stat(testFile)
		require.NoError(t, err)
		assert.Equal(t, os.FileMode(0600), info.Mode().Perm())
	})
}

func TestFileActivities_FileDelete(t *testing.T) {
	fa := &FileActivities{}
	ctx := context.Background()

	t.Run("delete existing file", func(t *testing.T) {
		tmpDir := t.TempDir()
		testFile := filepath.Join(tmpDir, "delete.txt")
		require.NoError(t, os.WriteFile(testFile, []byte("content"), 0644))

		input := FileDeleteInput{
			Path: testFile,
		}

		err := fa.FileDelete(ctx, input)
		require.NoError(t, err)

		// Verify file is gone
		_, err = os.Stat(testFile)
		assert.True(t, os.IsNotExist(err))
	})

	t.Run("idempotent - delete non-existent file", func(t *testing.T) {
		input := FileDeleteInput{
			Path: "/nonexistent/file.txt",
		}

		// Should succeed even though file doesn't exist
		err := fa.FileDelete(ctx, input)
		require.NoError(t, err)
	})

	t.Run("delete directory recursively", func(t *testing.T) {
		tmpDir := t.TempDir()
		dirToDelete := filepath.Join(tmpDir, "to-delete")
		require.NoError(t, os.MkdirAll(filepath.Join(dirToDelete, "nested"), 0755))
		require.NoError(t, os.WriteFile(filepath.Join(dirToDelete, "file.txt"), []byte("content"), 0644))

		input := FileDeleteInput{
			Path:      dirToDelete,
			Recursive: true,
		}

		err := fa.FileDelete(ctx, input)
		require.NoError(t, err)

		// Verify directory is gone
		_, err = os.Stat(dirToDelete)
		assert.True(t, os.IsNotExist(err))
	})

	t.Run("fail to delete directory without recursive", func(t *testing.T) {
		tmpDir := t.TempDir()
		dirToDelete := filepath.Join(tmpDir, "dir")
		require.NoError(t, os.Mkdir(dirToDelete, 0755))

		input := FileDeleteInput{
			Path:      dirToDelete,
			Recursive: false,
		}

		err := fa.FileDelete(ctx, input)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "directory")
	})
}

func TestFileActivities_FileList(t *testing.T) {
	fa := &FileActivities{}
	ctx := context.Background()

	setupTestDir := func(t *testing.T) string {
		t.Helper()
		tmpDir := t.TempDir()

		// Create test structure
		require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "file1.txt"), []byte("content"), 0644))
		require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "file2.go"), []byte("code"), 0644))
		require.NoError(t, os.Mkdir(filepath.Join(tmpDir, "subdir"), 0755))
		require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "subdir", "nested.txt"), []byte("nested"), 0644))

		return tmpDir
	}

	t.Run("list directory non-recursive", func(t *testing.T) {
		tmpDir := setupTestDir(t)

		input := FileListInput{
			Path:      tmpDir,
			Recursive: false,
		}

		output, err := fa.FileList(ctx, input)
		require.NoError(t, err)
		assert.Equal(t, 3, output.Count) // file1.txt, file2.go, subdir
		assert.Len(t, output.Files, 3)

		// Check files are present
		names := make([]string, len(output.Files))
		for i, f := range output.Files {
			names[i] = f.Name
		}
		assert.Contains(t, names, "file1.txt")
		assert.Contains(t, names, "file2.go")
		assert.Contains(t, names, "subdir")
	})

	t.Run("list directory recursively", func(t *testing.T) {
		tmpDir := setupTestDir(t)

		input := FileListInput{
			Path:      tmpDir,
			Recursive: true,
		}

		output, err := fa.FileList(ctx, input)
		require.NoError(t, err)
		assert.Equal(t, 4, output.Count) // file1.txt, file2.go, subdir, subdir/nested.txt
		assert.Len(t, output.Files, 4)
	})

	t.Run("list with glob pattern", func(t *testing.T) {
		tmpDir := setupTestDir(t)

		input := FileListInput{
			Path:      tmpDir,
			Pattern:   "*.txt",
			Recursive: false,
		}

		output, err := fa.FileList(ctx, input)
		require.NoError(t, err)
		assert.Equal(t, 1, output.Count) // only file1.txt
		assert.Equal(t, "file1.txt", output.Files[0].Name)
	})

	t.Run("error on non-existent directory", func(t *testing.T) {
		input := FileListInput{
			Path: "/nonexistent/dir",
		}

		_, err := fa.FileList(ctx, input)
		assert.Error(t, err)
	})

	t.Run("error on file instead of directory", func(t *testing.T) {
		tmpDir := t.TempDir()
		testFile := filepath.Join(tmpDir, "file.txt")
		require.NoError(t, os.WriteFile(testFile, []byte("content"), 0644))

		input := FileListInput{
			Path: testFile,
		}

		_, err := fa.FileList(ctx, input)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not a directory")
	})
}

func TestFileActivities_FileCopy(t *testing.T) {
	fa := &FileActivities{}
	ctx := context.Background()

	t.Run("copy file", func(t *testing.T) {
		tmpDir := t.TempDir()
		source := filepath.Join(tmpDir, "source.txt")
		dest := filepath.Join(tmpDir, "dest.txt")
		content := "Copy me"

		require.NoError(t, os.WriteFile(source, []byte(content), 0644))

		input := FileCopyInput{
			Source:      source,
			Destination: dest,
		}

		output, err := fa.FileCopy(ctx, input)
		require.NoError(t, err)
		assert.Equal(t, source, output.Source)
		assert.Equal(t, dest, output.Destination)
		assert.Equal(t, int64(len(content)), output.BytesCopied)
		assert.False(t, output.Overwritten)

		// Verify copy
		data, err := os.ReadFile(dest)
		require.NoError(t, err)
		assert.Equal(t, content, string(data))
	})

	t.Run("overwrite existing file", func(t *testing.T) {
		tmpDir := t.TempDir()
		source := filepath.Join(tmpDir, "source.txt")
		dest := filepath.Join(tmpDir, "dest.txt")

		require.NoError(t, os.WriteFile(source, []byte("new"), 0644))
		require.NoError(t, os.WriteFile(dest, []byte("old"), 0644))

		input := FileCopyInput{
			Source:      source,
			Destination: dest,
			Overwrite:   true,
		}

		output, err := fa.FileCopy(ctx, input)
		require.NoError(t, err)
		assert.True(t, output.Overwritten)

		// Verify overwrite
		data, err := os.ReadFile(dest)
		require.NoError(t, err)
		assert.Equal(t, "new", string(data))
	})

	t.Run("fail to overwrite without flag", func(t *testing.T) {
		tmpDir := t.TempDir()
		source := filepath.Join(tmpDir, "source.txt")
		dest := filepath.Join(tmpDir, "dest.txt")

		require.NoError(t, os.WriteFile(source, []byte("new"), 0644))
		require.NoError(t, os.WriteFile(dest, []byte("old"), 0644))

		input := FileCopyInput{
			Source:      source,
			Destination: dest,
			Overwrite:   false,
		}

		_, err := fa.FileCopy(ctx, input)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Overwrite=false")
	})

	t.Run("create parent directories", func(t *testing.T) {
		tmpDir := t.TempDir()
		source := filepath.Join(tmpDir, "source.txt")
		dest := filepath.Join(tmpDir, "nested", "dir", "dest.txt")

		require.NoError(t, os.WriteFile(source, []byte("content"), 0644))

		input := FileCopyInput{
			Source:      source,
			Destination: dest,
			CreateDir:   true,
		}

		_, err := fa.FileCopy(ctx, input)
		require.NoError(t, err)

		// Verify file was created
		_, err = os.Stat(dest)
		require.NoError(t, err)
	})
}

func TestFileActivities_FileMove(t *testing.T) {
	fa := &FileActivities{}
	ctx := context.Background()

	t.Run("move file", func(t *testing.T) {
		tmpDir := t.TempDir()
		source := filepath.Join(tmpDir, "source.txt")
		dest := filepath.Join(tmpDir, "dest.txt")
		content := "Move me"

		require.NoError(t, os.WriteFile(source, []byte(content), 0644))

		input := FileMoveInput{
			Source:      source,
			Destination: dest,
		}

		err := fa.FileMove(ctx, input)
		require.NoError(t, err)

		// Verify source is gone
		_, err = os.Stat(source)
		assert.True(t, os.IsNotExist(err))

		// Verify destination exists
		data, err := os.ReadFile(dest)
		require.NoError(t, err)
		assert.Equal(t, content, string(data))
	})

	t.Run("idempotent - already moved", func(t *testing.T) {
		tmpDir := t.TempDir()
		source := filepath.Join(tmpDir, "source.txt")
		dest := filepath.Join(tmpDir, "dest.txt")

		require.NoError(t, os.WriteFile(dest, []byte("content"), 0644))

		input := FileMoveInput{
			Source:      source,
			Destination: dest,
		}

		// Should succeed even though source doesn't exist but dest does
		err := fa.FileMove(ctx, input)
		require.NoError(t, err)
	})

	t.Run("overwrite existing destination", func(t *testing.T) {
		tmpDir := t.TempDir()
		source := filepath.Join(tmpDir, "source.txt")
		dest := filepath.Join(tmpDir, "dest.txt")

		require.NoError(t, os.WriteFile(source, []byte("new"), 0644))
		require.NoError(t, os.WriteFile(dest, []byte("old"), 0644))

		input := FileMoveInput{
			Source:      source,
			Destination: dest,
			Overwrite:   true,
		}

		err := fa.FileMove(ctx, input)
		require.NoError(t, err)

		// Verify content
		data, err := os.ReadFile(dest)
		require.NoError(t, err)
		assert.Equal(t, "new", string(data))
	})

	t.Run("create parent directories", func(t *testing.T) {
		tmpDir := t.TempDir()
		source := filepath.Join(tmpDir, "source.txt")
		dest := filepath.Join(tmpDir, "nested", "dir", "dest.txt")

		require.NoError(t, os.WriteFile(source, []byte("content"), 0644))

		input := FileMoveInput{
			Source:      source,
			Destination: dest,
			CreateDir:   true,
		}

		err := fa.FileMove(ctx, input)
		require.NoError(t, err)

		// Verify file was moved
		_, err = os.Stat(dest)
		require.NoError(t, err)
	})
}

func TestFileActivities_FileExists(t *testing.T) {
	fa := &FileActivities{}
	ctx := context.Background()

	t.Run("file exists", func(t *testing.T) {
		tmpDir := t.TempDir()
		testFile := filepath.Join(tmpDir, "exists.txt")
		content := "I exist"
		require.NoError(t, os.WriteFile(testFile, []byte(content), 0644))

		input := FileExistsInput{
			Path: testFile,
		}

		output, err := fa.FileExists(ctx, input)
		require.NoError(t, err)
		assert.True(t, output.Exists)
		assert.False(t, output.IsDir)
		assert.Equal(t, int64(len(content)), output.Size)
	})

	t.Run("directory exists", func(t *testing.T) {
		tmpDir := t.TempDir()

		input := FileExistsInput{
			Path: tmpDir,
		}

		output, err := fa.FileExists(ctx, input)
		require.NoError(t, err)
		assert.True(t, output.Exists)
		assert.True(t, output.IsDir)
		assert.Equal(t, int64(0), output.Size)
	})

	t.Run("file does not exist", func(t *testing.T) {
		input := FileExistsInput{
			Path: "/nonexistent/file.txt",
		}

		output, err := fa.FileExists(ctx, input)
		require.NoError(t, err)
		assert.False(t, output.Exists)
		assert.False(t, output.IsDir)
		assert.Equal(t, int64(0), output.Size)
	})
}
