package ops

import (
	pb "github.com/moby/buildkit/solver/pb"

	"github.com/kasuboski/luakit/pkg/dag"
)

type Mount struct {
	mount *pb.Mount
}

type CacheOptions struct {
	ID      string
	Sharing string
}

type SecretOptions struct {
	ID       string
	UID      uint32
	GID      uint32
	Mode     uint32
	Optional bool
}

type SSHOptions struct {
	Dest     string
	ID       string
	UID      uint32
	GID      uint32
	Mode     uint32
	Optional bool
}

type TmpfsOptions struct {
	Size int64
}

type BindOptions struct {
	Selector string
	Readonly bool
}

func CacheMount(dest string, opts *CacheOptions) *Mount {
	mountType := pb.MountType_CACHE
	cacheOpt := &pb.CacheOpt{}

	if opts != nil {
		if opts.ID != "" {
			cacheOpt.ID = opts.ID
		}
		if opts.Sharing != "" {
			switch opts.Sharing {
			case "shared":
				cacheOpt.Sharing = pb.CacheSharingOpt_SHARED
			case "private":
				cacheOpt.Sharing = pb.CacheSharingOpt_PRIVATE
			case "locked":
				cacheOpt.Sharing = pb.CacheSharingOpt_LOCKED
			}
		}
	}

	return &Mount{
		mount: &pb.Mount{
			Dest:      dest,
			MountType: mountType,
			CacheOpt:  cacheOpt,
		},
	}
}

func SecretMount(dest string, opts *SecretOptions) *Mount {
	mountType := pb.MountType_SECRET
	secretOpt := &pb.SecretOpt{
		Uid:  0,
		Gid:  0,
		Mode: 0400,
	}

	if opts != nil {
		if opts.ID != "" {
			secretOpt.ID = opts.ID
		}
		if opts.UID != 0 {
			secretOpt.Uid = opts.UID
		}
		if opts.GID != 0 {
			secretOpt.Gid = opts.GID
		}
		if opts.Mode != 0 {
			secretOpt.Mode = opts.Mode
		}
		secretOpt.Optional = opts.Optional
	}

	return &Mount{
		mount: &pb.Mount{
			Dest:      dest,
			MountType: mountType,
			SecretOpt: secretOpt,
		},
	}
}

func SSHMount(opts *SSHOptions) *Mount {
	mountType := pb.MountType_SSH
	sshOpt := &pb.SSHOpt{
		Uid:  0,
		Gid:  0,
		Mode: 0600,
	}

	dest := "/run/ssh"
	if opts != nil {
		if opts.Dest != "" {
			dest = opts.Dest
		}
		if opts.ID != "" {
			sshOpt.ID = opts.ID
		}
		if opts.UID != 0 {
			sshOpt.Uid = opts.UID
		}
		if opts.GID != 0 {
			sshOpt.Gid = opts.GID
		}
		if opts.Mode != 0 {
			sshOpt.Mode = opts.Mode
		}
		sshOpt.Optional = opts.Optional
	}

	return &Mount{
		mount: &pb.Mount{
			Dest:      dest,
			MountType: mountType,
			SSHOpt:    sshOpt,
		},
	}
}

func TmpfsMount(dest string, opts *TmpfsOptions) *Mount {
	mountType := pb.MountType_TMPFS
	tmpfsOpt := &pb.TmpfsOpt{}

	if opts != nil && opts.Size > 0 {
		tmpfsOpt.Size = opts.Size
	}

	return &Mount{
		mount: &pb.Mount{
			Dest:      dest,
			MountType: mountType,
			TmpfsOpt:  tmpfsOpt,
		},
	}
}

func BindMount(state *dag.State, dest string, opts *BindOptions) *Mount {
	mountType := pb.MountType_BIND
	input := int64(len(state.Op().Inputs()))

	mount := &pb.Mount{
		Input:     input,
		Dest:      dest,
		MountType: mountType,
		Readonly:  true,
	}

	if opts != nil {
		if opts.Selector != "" {
			mount.Selector = opts.Selector
		}
		mount.Readonly = opts.Readonly
	}

	return &Mount{
		mount: mount,
	}
}

func (m *Mount) ToPB() *pb.Mount {
	return m.mount
}
