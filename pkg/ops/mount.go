package ops

import (
	pb "github.com/moby/buildkit/solver/pb"

	"github.com/kasuboski/luakit/pkg/dag"
)

type Mount struct {
	state *dag.State
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

func applyFileOpt(opt any, uid, gid, mode uint32, optional bool) {
	switch o := opt.(type) {
	case *pb.SecretOpt:
		if uid != 0 {
			o.Uid = uid
		}
		if gid != 0 {
			o.Gid = gid
		}
		if mode != 0 {
			o.Mode = mode
		}
		o.Optional = optional
	case *pb.SSHOpt:
		if uid != 0 {
			o.Uid = uid
		}
		if gid != 0 {
			o.Gid = gid
		}
		if mode != 0 {
			o.Mode = mode
		}
		o.Optional = optional
	}
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
		applyFileOpt(secretOpt, opts.UID, opts.GID, opts.Mode, opts.Optional)
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
		applyFileOpt(sshOpt, opts.UID, opts.GID, opts.Mode, opts.Optional)
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

	mountPb := &pb.Mount{
		Input:     input,
		Dest:      dest,
		MountType: mountType,
		Readonly:  true,
	}

	if opts != nil {
		if opts.Selector != "" {
			mountPb.Selector = opts.Selector
		}
		mountPb.Readonly = opts.Readonly
	}

	return &Mount{
		mount: mountPb,
		state: state,
	}
}

func (m *Mount) ToPB() *pb.Mount {
	return m.mount
}
