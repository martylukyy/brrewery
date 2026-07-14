package selfupdate

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
)

// maxExtractFileSize caps a single extracted file (the binary is the largest
// entry at a few tens of MB), bounding decompression-bomb blowup.
const maxExtractFileSize = 1 << 30 // 1 GiB

// extractTarGz unpacks the release archive into dest. Only directories and
// regular files are accepted; entries that would escape dest (absolute paths,
// "..", symlinks) fail the whole extraction.
func extractTarGz(archivePath, dest string) error {
	f, err := os.Open(archivePath)
	if err != nil {
		return fmt.Errorf("open archive: %w", err)
	}
	defer f.Close()

	gz, err := gzip.NewReader(f)
	if err != nil {
		return fmt.Errorf("read gzip: %w", err)
	}
	defer gz.Close()

	if err := os.MkdirAll(dest, 0o750); err != nil {
		return fmt.Errorf("create extract dir: %w", err)
	}

	reader := tar.NewReader(gz)
	for {
		header, err := reader.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return fmt.Errorf("read archive: %w", err)
		}
		if err := extractEntry(header, reader, dest); err != nil {
			return err
		}
	}
}

func extractEntry(header *tar.Header, reader io.Reader, dest string) error {
	name := filepath.Clean(header.Name)
	if !filepath.IsLocal(name) {
		return fmt.Errorf("archive entry escapes extraction dir: %s", header.Name)
	}
	target := filepath.Join(dest, name)

	switch header.Typeflag {
	case tar.TypeDir:
		if err := os.MkdirAll(target, header.FileInfo().Mode().Perm()|0o700); err != nil {
			return fmt.Errorf("create dir %s: %w", name, err)
		}
		return nil
	case tar.TypeReg:
		return extractFile(name, target, header.FileInfo().Mode().Perm(), reader)
	default:
		return fmt.Errorf("unsupported archive entry type %q for %s", header.Typeflag, header.Name)
	}
}

func extractFile(name, target string, mode os.FileMode, reader io.Reader) error {
	if err := os.MkdirAll(filepath.Dir(target), 0o750); err != nil {
		return fmt.Errorf("create dir for %s: %w", name, err)
	}
	out, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, mode)
	if err != nil {
		return fmt.Errorf("create file %s: %w", name, err)
	}
	written, err := io.Copy(out, io.LimitReader(reader, maxExtractFileSize+1))
	if err != nil {
		_ = out.Close()
		return fmt.Errorf("write file %s: %w", name, err)
	}
	if written > maxExtractFileSize {
		_ = out.Close()
		return fmt.Errorf("archive entry %s exceeds the %d byte extraction limit", name, maxExtractFileSize)
	}
	if err := out.Close(); err != nil {
		return fmt.Errorf("close file %s: %w", name, err)
	}
	return nil
}

// copyFile copies src to dst with the given mode, truncating any existing file.
func copyFile(src, dst string, mode os.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("open %s: %w", src, err)
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, mode)
	if err != nil {
		return fmt.Errorf("create %s: %w", dst, err)
	}
	if _, err := io.Copy(out, in); err != nil {
		_ = out.Close()
		return fmt.Errorf("copy to %s: %w", dst, err)
	}
	if err := out.Close(); err != nil {
		return fmt.Errorf("close %s: %w", dst, err)
	}
	return os.Chmod(dst, mode)
}

// copyTree copies a directory of regular files, preserving file modes.
func copyTree(src, dst string) error {
	return filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)

		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		if !d.Type().IsRegular() {
			return fmt.Errorf("unsupported file type in %s", path)
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		return copyFile(path, target, info.Mode().Perm())
	})
}

// swapDir replaces live with a copy of src using sibling renames, so the live
// tree is never left half-populated: copy to live.new, move live aside to
// live.old, rename live.new into place, then drop live.old. (install.sh
// instead wipes the live dir before copying, leaving an empty-dir window.)
func swapDir(src, live string) error {
	newDir := live + ".new"
	oldDir := live + ".old"
	_ = os.RemoveAll(newDir)
	_ = os.RemoveAll(oldDir)

	if err := os.MkdirAll(filepath.Dir(live), 0o755); err != nil {
		return fmt.Errorf("create parent of %s: %w", live, err)
	}
	if err := copyTree(src, newDir); err != nil {
		_ = os.RemoveAll(newDir)
		return fmt.Errorf("stage %s: %w", newDir, err)
	}

	liveExists := false
	if _, err := os.Stat(live); err == nil {
		liveExists = true
		if err := os.Rename(live, oldDir); err != nil {
			_ = os.RemoveAll(newDir)
			return fmt.Errorf("move aside %s: %w", live, err)
		}
	}
	if err := os.Rename(newDir, live); err != nil {
		if liveExists {
			_ = os.Rename(oldDir, live)
		}
		_ = os.RemoveAll(newDir)
		return fmt.Errorf("activate %s: %w", live, err)
	}
	_ = os.RemoveAll(oldDir)
	return nil
}
