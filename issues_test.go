package zipologger

import (
	"encoding/base64"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/MasterDimmy/zipologger/enc"
)

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

// TestLazyDirCreation verifies that NewLogger does NOT create the target
// directory until the first write actually happens (issues #1/#2).
func TestLazyDirCreation(t *testing.T) {
	SetAlsoToStdout(false)

	name := filepath.Join(t.TempDir(), "nested", "sub", "app.log")
	if dirExists(filepath.Dir(name)) {
		t.Fatalf("directory should not exist before any write: %s", filepath.Dir(name))
	}

	l := NewLogger(name, 1, 1, 1, false)
	// No write yet -> directory must still not exist.
	if dirExists(filepath.Dir(name)) {
		t.Fatalf("directory was created eagerly on NewLogger: %s", filepath.Dir(name))
	}

	l.Print("first write triggers directory creation")
	l.Wait()
	l.Close()

	if _, err := os.Stat(name); err != nil {
		t.Fatalf("log file was not created after write: %v", err)
	}
	if !dirExists(filepath.Dir(name)) {
		t.Fatalf("directory was not created on first write: %s", filepath.Dir(name))
	}
}

// TestGlobalEncryptionWorks verifies that SetGlobalEncryption actually enables
// encryption for loggers that do not set a per-logger key (issue #3).
func TestGlobalEncryptionWorks(t *testing.T) {
	SetAlsoToStdout(false)
	defer func() {
		mainGlobalEncryptor.m.Lock()
		mainGlobalEncryptor.key = nil
		mainGlobalEncryptor.m.Unlock()
	}()

	key := enc.NewKey()
	if !SetGlobalEncryption(key.EncryptionKey()) {
		t.Fatalf("SetGlobalEncryption returned false for a valid key")
	}

	name := filepath.Join(t.TempDir(), "enc", "global.log")
	l := NewLogger(name, 1, 1, 1, false)

	const msg = "global-enc-test-message"
	l.Print(msg)
	l.Wait()
	l.Close()

	data, err := os.ReadFile(name)
	if err != nil {
		t.Fatalf("cannot read log file: %v", err)
	}
	if strings.Contains(string(data), msg) {
		t.Fatalf("message was stored in plaintext, global encryption did not apply: %q", string(data))
	}

	dec := enc.NewDecryptKey(key.DecryptionKey())
	if dec == nil {
		t.Fatalf("invalid decryption key")
	}
	for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		if line == "" {
			continue
		}
		raw, err := base64.RawStdEncoding.DecodeString(line)
		if err != nil {
			t.Fatalf("line is not valid base64 (encryption missing?): %v -> %q", err, line)
		}
		plain, err := dec.Decrypt(raw)
		if err != nil {
			t.Fatalf("decryption failed: %v", err)
		}
		if !strings.Contains(string(plain), msg) {
			t.Fatalf("decrypted content mismatch: %q", string(plain))
		}
	}
}

// TestSetEncryptionKeyError verifies that an invalid key returns an error
// instead of panicking (issue #6).
func TestSetEncryptionKeyError(t *testing.T) {
	l := &Logger{}

	if _, err := l.SetEncryptionKey("not-a-valid-hex-key"); err == nil {
		t.Fatalf("expected error for invalid encryption key, got nil")
	}

	key := enc.NewKey()
	if _, err := l.SetEncryptionKey(key.EncryptionKey()); err != nil {
		t.Fatalf("expected no error for valid key, got %v", err)
	}
}

// TestRotationSmoke verifies the real writer (zlog) is used and rotation works
// without the previously redundant second file handle (issues #4/#5).
func TestRotationSmoke(t *testing.T) {
	SetAlsoToStdout(false)

	name := filepath.Join(t.TempDir(), "rot", "test.log")
	l := NewLogger(name, 1, 5, 1, false)

	l.Print("seed line")
	l.Wait()

	if l.zlog == nil {
		t.Fatalf("zlog writer was not initialized after first write")
	}

	if err := l.zlog.Rotate(); err != nil {
		t.Fatalf("rotation failed: %v", err)
	}
	l.Wait()

	entries, err := os.ReadDir(filepath.Dir(name))
	if err != nil {
		t.Fatalf("cannot read log dir: %v", err)
	}
	foundZip := false
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".zip") {
			foundZip = true
			break
		}
	}
	if !foundZip {
		t.Fatalf("rotation did not produce a backup .zip file")
	}
	l.Close()
}

// TestSavePanicToFileCreatesTargetDir verifies that savePanicToFile actually
// writes the panic log file (issue #7). Previously it could return an empty
// string (silently losing the panic log) when the target directory did not
// exist because Mkdir was called on the wrong (CWD) directory.
func TestSavePanicToFileCreatesTargetDir(t *testing.T) {
	result := savePanicToFile("panic test content")
	if result == "" {
		t.Fatalf("savePanicToFile returned empty string (file creation failed)")
	}

	// The panic log must have been written to either the executable directory
	// or the current working directory (fallback). Verify a matching file exists.
	st, _ := filepath.Abs(os.Args[0])
	base := filepath.Base(os.Args[0])
	candidates := []string{
		filepath.Join(filepath.Dir(st), "logs"),
		"logs",
	}
	found := false
	for _, dir := range candidates {
		matches, _ := filepath.Glob(filepath.Join(dir, "panic_"+base+"*"))
		if len(matches) > 0 {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("panic log file was not created in any candidate directory")
	}
}
