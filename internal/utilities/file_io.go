package utilities

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

func MakeDirParent(destFilePath string) (*os.File, error) {
	_, err := os.Stat(destFilePath)
	if errors.Is(err, os.ErrNotExist) {
		err = os.MkdirAll(filepath.Dir(destFilePath), os.ModePerm)
		if err != nil {
			return nil, err
		}
	}
	destFile, err := os.Create(destFilePath)
	if err != nil {
		return nil, err
	}
	return destFile, nil
}

func ListFilesWithExt(root, ext string, recursive bool) ([]string, error) {
	if root == "" {
		return nil, errors.New("root directory is empty")
	}
	if ext == "" {
		return nil, errors.New("extension is empty")
	}
	if !strings.HasPrefix(ext, ".") {
		ext = "." + ext
	}
	ext = strings.ToLower(ext)

	info, err := os.Stat(root)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, errors.New("root is not a directory")
	}

	var out []string

	if !recursive {
		entries, err := os.ReadDir(root)
		if err != nil {
			return nil, err
		}
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			name := e.Name()
			if strings.EqualFold(filepath.Ext(name), ext) {
				abs := filepath.Join(root, name)
				abs, _ = filepath.Abs(abs)
				out = append(out, abs)
			}
		}
		return out, nil
	}

	// Recursive: use WalkDir
	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		if strings.EqualFold(filepath.Ext(d.Name()), ext) {
			abs, _ := filepath.Abs(path)
			out = append(out, abs)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}
