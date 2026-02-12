package ops

import (
	"testing"

	pb "github.com/moby/buildkit/solver/pb"
)

func TestCacheMount(t *testing.T) {
	mount := CacheMount("/cache", &CacheOptions{
		ID:      "mycache",
		Sharing: "shared",
	})

	if mount == nil {
		t.Fatal("Expected non-nil mount")
	}

	pbMount := mount.ToPB()
	if pbMount.Dest != "/cache" {
		t.Errorf("Expected dest '/cache', got '%s'", pbMount.Dest)
	}

	if pbMount.MountType != pb.MountType_CACHE {
		t.Errorf("Expected mount type CACHE, got %v", pbMount.MountType)
	}

	if pbMount.CacheOpt.ID != "mycache" {
		t.Errorf("Expected cache ID 'mycache', got '%s'", pbMount.CacheOpt.ID)
	}

	if pbMount.CacheOpt.Sharing != pb.CacheSharingOpt_SHARED {
		t.Errorf("Expected sharing SHARED, got %v", pbMount.CacheOpt.Sharing)
	}
}

func TestCacheMountWithNoOptions(t *testing.T) {
	mount := CacheMount("/cache", nil)

	if mount == nil {
		t.Fatal("Expected non-nil mount")
	}

	pbMount := mount.ToPB()
	if pbMount.Dest != "/cache" {
		t.Errorf("Expected dest '/cache', got '%s'", pbMount.Dest)
	}

	if pbMount.MountType != pb.MountType_CACHE {
		t.Errorf("Expected mount type CACHE, got %v", pbMount.MountType)
	}
}

func TestSecretMount(t *testing.T) {
	mount := SecretMount("/run/secrets/secret", &SecretOptions{
		ID:       "mysecret",
		UID:      1000,
		GID:      1000,
		Mode:     0600,
		Optional: true,
	})

	if mount == nil {
		t.Fatal("Expected non-nil mount")
	}

	pbMount := mount.ToPB()
	if pbMount.Dest != "/run/secrets/secret" {
		t.Errorf("Expected dest '/run/secrets/secret', got '%s'", pbMount.Dest)
	}

	if pbMount.MountType != pb.MountType_SECRET {
		t.Errorf("Expected mount type SECRET, got %v", pbMount.MountType)
	}

	if pbMount.SecretOpt.ID != "mysecret" {
		t.Errorf("Expected secret ID 'mysecret', got '%s'", pbMount.SecretOpt.ID)
	}

	if pbMount.SecretOpt.Uid != 1000 {
		t.Errorf("Expected uid 1000, got %d", pbMount.SecretOpt.Uid)
	}

	if pbMount.SecretOpt.Gid != 1000 {
		t.Errorf("Expected gid 1000, got %d", pbMount.SecretOpt.Gid)
	}

	if pbMount.SecretOpt.Mode != 0600 {
		t.Errorf("Expected mode 0600, got %d", pbMount.SecretOpt.Mode)
	}

	if !pbMount.SecretOpt.Optional {
		t.Error("Expected optional to be true")
	}
}

func TestSecretMountDefaults(t *testing.T) {
	mount := SecretMount("/run/secrets/secret", nil)

	if mount == nil {
		t.Fatal("Expected non-nil mount")
	}

	pbMount := mount.ToPB()
	if pbMount.SecretOpt.Uid != 0 {
		t.Errorf("Expected default uid 0, got %d", pbMount.SecretOpt.Uid)
	}

	if pbMount.SecretOpt.Gid != 0 {
		t.Errorf("Expected default gid 0, got %d", pbMount.SecretOpt.Gid)
	}

	if pbMount.SecretOpt.Mode != 0400 {
		t.Errorf("Expected default mode 0400, got %d", pbMount.SecretOpt.Mode)
	}
}

func TestSSHHMount(t *testing.T) {
	mount := SSHMount(&SSHOptions{
		Dest:     "/custom/ssh",
		ID:       "myssh",
		UID:      1000,
		GID:      1000,
		Mode:     0644,
		Optional: false,
	})

	if mount == nil {
		t.Fatal("Expected non-nil mount")
	}

	pbMount := mount.ToPB()
	if pbMount.Dest != "/custom/ssh" {
		t.Errorf("Expected dest '/custom/ssh', got '%s'", pbMount.Dest)
	}

	if pbMount.MountType != pb.MountType_SSH {
		t.Errorf("Expected mount type SSH, got %v", pbMount.MountType)
	}

	if pbMount.SSHOpt.ID != "myssh" {
		t.Errorf("Expected ssh ID 'myssh', got '%s'", pbMount.SSHOpt.ID)
	}

	if pbMount.SSHOpt.Uid != 1000 {
		t.Errorf("Expected uid 1000, got %d", pbMount.SSHOpt.Uid)
	}

	if pbMount.SSHOpt.Gid != 1000 {
		t.Errorf("Expected gid 1000, got %d", pbMount.SSHOpt.Gid)
	}

	if pbMount.SSHOpt.Mode != 0644 {
		t.Errorf("Expected mode 0644, got %d", pbMount.SSHOpt.Mode)
	}

	if pbMount.SSHOpt.Optional {
		t.Error("Expected optional to be false")
	}
}

func TestSSHMountDefaults(t *testing.T) {
	mount := SSHMount(nil)

	if mount == nil {
		t.Fatal("Expected non-nil mount")
	}

	pbMount := mount.ToPB()
	if pbMount.Dest != "/run/ssh" {
		t.Errorf("Expected default dest '/run/ssh', got '%s'", pbMount.Dest)
	}

	if pbMount.SSHOpt.Uid != 0 {
		t.Errorf("Expected default uid 0, got %d", pbMount.SSHOpt.Uid)
	}

	if pbMount.SSHOpt.Gid != 0 {
		t.Errorf("Expected default gid 0, got %d", pbMount.SSHOpt.Gid)
	}

	if pbMount.SSHOpt.Mode != 0600 {
		t.Errorf("Expected default mode 0600, got %d", pbMount.SSHOpt.Mode)
	}
}

func TestTmpfsMount(t *testing.T) {
	mount := TmpfsMount("/tmp", &TmpfsOptions{
		Size: 1024 * 1024 * 1024,
	})

	if mount == nil {
		t.Fatal("Expected non-nil mount")
	}

	pbMount := mount.ToPB()
	if pbMount.Dest != "/tmp" {
		t.Errorf("Expected dest '/tmp', got '%s'", pbMount.Dest)
	}

	if pbMount.MountType != pb.MountType_TMPFS {
		t.Errorf("Expected mount type TMPFS, got %v", pbMount.MountType)
	}

	if pbMount.TmpfsOpt.Size != 1024*1024*1024 {
		t.Errorf("Expected size 1073741824, got %d", pbMount.TmpfsOpt.Size)
	}
}

func TestTmpfsMountWithNoOptions(t *testing.T) {
	mount := TmpfsMount("/tmp", nil)

	if mount == nil {
		t.Fatal("Expected non-nil mount")
	}

	pbMount := mount.ToPB()
	if pbMount.Dest != "/tmp" {
		t.Errorf("Expected dest '/tmp', got '%s'", pbMount.Dest)
	}

	if pbMount.MountType != pb.MountType_TMPFS {
		t.Errorf("Expected mount type TMPFS, got %v", pbMount.MountType)
	}
}

func TestBindMount(t *testing.T) {
	sourceOp := NewSourceOp("scratch", nil)
	sourceState := NewSourceState(sourceOp, "test.lua", 10)

	mount := BindMount(sourceState, "/bind/target", &BindOptions{
		Selector: "/specific/path",
		Readonly: false,
	})

	if mount == nil {
		t.Fatal("Expected non-nil mount")
	}

	pbMount := mount.ToPB()
	if pbMount.Dest != "/bind/target" {
		t.Errorf("Expected dest '/bind/target', got '%s'", pbMount.Dest)
	}

	if pbMount.MountType != pb.MountType_BIND {
		t.Errorf("Expected mount type BIND, got %v", pbMount.MountType)
	}

	if pbMount.Selector != "/specific/path" {
		t.Errorf("Expected selector '/specific/path', got '%s'", pbMount.Selector)
	}

	if pbMount.Readonly {
		t.Error("Expected readonly to be false")
	}
}

func TestBindMountDefaults(t *testing.T) {
	sourceOp := NewSourceOp("scratch", nil)
	sourceState := NewSourceState(sourceOp, "test.lua", 10)

	mount := BindMount(sourceState, "/bind/target", nil)

	if mount == nil {
		t.Fatal("Expected non-nil mount")
	}

	pbMount := mount.ToPB()
	if pbMount.MountType != pb.MountType_BIND {
		t.Errorf("Expected mount type BIND, got %v", pbMount.MountType)
	}

	if !pbMount.Readonly {
		t.Error("Expected default readonly to be true")
	}
}

func TestExecOpWithMounts(t *testing.T) {
	cacheMount := CacheMount("/cache", &CacheOptions{ID: "mycache"})
	secretMount := SecretMount("/run/secrets/secret", &SecretOptions{ID: "mysecret"})
	tmpfsMount := TmpfsMount("/tmp", &TmpfsOptions{Size: 1024 * 1024})

	opts := &ExecOptions{
		Env:    []string{"TEST=1"},
		Cwd:    "/app",
		User:   "nobody",
		Mounts: []*Mount{cacheMount, secretMount, tmpfsMount},
	}

	op := NewExecOp([]string{"echo", "hello"}, opts)

	if len(op.Mounts) != 3 {
		t.Errorf("Expected 3 mounts, got %d", len(op.Mounts))
	}

	if op.Mounts[0].Dest != "/cache" {
		t.Errorf("Expected first mount dest '/cache', got '%s'", op.Mounts[0].Dest)
	}

	if op.Mounts[1].Dest != "/run/secrets/secret" {
		t.Errorf("Expected second mount dest '/run/secrets/secret', got '%s'", op.Mounts[1].Dest)
	}

	if op.Mounts[2].Dest != "/tmp" {
		t.Errorf("Expected third mount dest '/tmp', got '%s'", op.Mounts[2].Dest)
	}
}

func TestExecOpWithNoMounts(t *testing.T) {
	opts := &ExecOptions{
		Env:  []string{"TEST=1"},
		Cwd:  "/app",
		User: "nobody",
	}

	op := NewExecOp([]string{"echo", "hello"}, opts)

	if len(op.Mounts) != 0 {
		t.Errorf("Expected 0 mounts, got %d", len(op.Mounts))
	}
}

func TestExecOpWithNilOptions(t *testing.T) {
	op := NewExecOp([]string{"echo", "hello"}, nil)

	if len(op.Mounts) != 0 {
		t.Errorf("Expected 0 mounts, got %d", len(op.Mounts))
	}
}
