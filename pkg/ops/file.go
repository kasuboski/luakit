package ops

import (
	pb "github.com/moby/buildkit/solver/pb"

	"github.com/kasuboski/luakit/pkg/dag"
)

type CopyOptions struct {
	Owner           *ChownOpt
	Mode            int32
	FollowSymlink   bool
	CreateDestPath  bool
	AllowWildcard   bool
	IncludePatterns []string
	ExcludePatterns []string
}

type MkdirOptions struct {
	Mode        int32
	MakeParents bool
	Owner       *ChownOpt
}

type MkfileOptions struct {
	Mode  int32
	Owner *ChownOpt
}

type RmOptions struct {
	AllowNotFound bool
	AllowWildcard bool
}

type ChownOpt struct {
	User  *UserOpt
	Group *UserOpt
}

type UserOpt struct {
	Name string
	ID   int64
}

func NewFileOp(actions []*pb.FileAction) *pb.FileOp {
	return &pb.FileOp{
		Actions: actions,
	}
}

func NewFileState(state *dag.State, op *pb.FileOp, luaFile string, luaLine int) *dag.State {
	pbOp := &pb.Op{
		Inputs: []*pb.Input{
			{
				Digest: string(state.Op().Digest()),
				Index:  int64(state.OutputIndex()),
			},
		},
		Op: &pb.Op_File{
			File: op,
		},
	}

	node := dag.NewOpNode(pbOp, luaFile, luaLine)
	node.AddInput(dag.NewEdge(state.Op(), state.OutputIndex()))

	return dag.NewState(node)
}

func NewCopyFileState(state *dag.State, fromState *dag.State, src, dest string, opts *CopyOptions, luaFile string, luaLine int) *dag.State {
	copyAction := &pb.FileActionCopy{
		Src:  src,
		Dest: dest,
	}

	if opts != nil {
		if opts.Owner != nil {
			copyAction.Owner = buildChownOpt(opts.Owner)
		}
		if opts.Mode != 0 {
			copyAction.Mode = opts.Mode
		}
		if opts.FollowSymlink {
			copyAction.FollowSymlink = opts.FollowSymlink
		}
		if opts.CreateDestPath {
			copyAction.CreateDestPath = opts.CreateDestPath
		}
		if opts.AllowWildcard {
			copyAction.AllowWildcard = opts.AllowWildcard
		}
		if len(opts.IncludePatterns) > 0 {
			copyAction.IncludePatterns = opts.IncludePatterns
		}
		if len(opts.ExcludePatterns) > 0 {
			copyAction.ExcludePatterns = opts.ExcludePatterns
		}
	}

	action := &pb.FileAction{
		Action: &pb.FileAction_Copy{
			Copy: copyAction,
		},
	}

	op := NewFileOp([]*pb.FileAction{action})
	pbOp := &pb.Op{
		Inputs: []*pb.Input{
			{
				Digest: string(state.Op().Digest()),
				Index:  int64(state.OutputIndex()),
			},
			{
				Digest: string(fromState.Op().Digest()),
				Index:  int64(fromState.OutputIndex()),
			},
		},
		Op: &pb.Op_File{
			File: op,
		},
	}

	node := dag.NewOpNode(pbOp, luaFile, luaLine)
	node.AddInput(dag.NewEdge(state.Op(), state.OutputIndex()))
	node.AddInput(dag.NewEdge(fromState.Op(), fromState.OutputIndex()))

	return dag.NewState(node)
}

func Copy(state *dag.State, fromState *dag.State, src, dest string, opts *CopyOptions, luaFile string, luaLine int) *dag.State {
	if src == "" || dest == "" {
		return nil
	}

	return NewCopyFileState(state, fromState, src, dest, opts, luaFile, luaLine)
}

func Mkdir(state *dag.State, path string, opts *MkdirOptions, luaFile string, luaLine int) *dag.State {
	if path == "" {
		return nil
	}

	mkdirAction := &pb.FileActionMkDir{
		Path: path,
	}

	if opts != nil {
		if opts.Mode != 0 {
			mkdirAction.Mode = opts.Mode
		}
		if opts.MakeParents {
			mkdirAction.MakeParents = opts.MakeParents
		}
		if opts.Owner != nil {
			mkdirAction.Owner = buildChownOpt(opts.Owner)
		}
	}

	action := &pb.FileAction{
		Action: &pb.FileAction_Mkdir{
			Mkdir: mkdirAction,
		},
	}

	op := NewFileOp([]*pb.FileAction{action})
	return NewFileState(state, op, luaFile, luaLine)
}

func Mkfile(state *dag.State, path, data string, opts *MkfileOptions, luaFile string, luaLine int) *dag.State {
	if path == "" {
		return nil
	}

	mkfileAction := &pb.FileActionMkFile{
		Path: path,
		Data: []byte(data),
	}

	if opts != nil {
		if opts.Mode != 0 {
			mkfileAction.Mode = opts.Mode
		}
		if opts.Owner != nil {
			mkfileAction.Owner = buildChownOpt(opts.Owner)
		}
	}

	action := &pb.FileAction{
		Action: &pb.FileAction_Mkfile{
			Mkfile: mkfileAction,
		},
	}

	op := NewFileOp([]*pb.FileAction{action})
	return NewFileState(state, op, luaFile, luaLine)
}

func Rm(state *dag.State, path string, opts *RmOptions, luaFile string, luaLine int) *dag.State {
	if path == "" {
		return nil
	}

	rmAction := &pb.FileActionRm{
		Path: path,
	}

	if opts != nil {
		if opts.AllowNotFound {
			rmAction.AllowNotFound = opts.AllowNotFound
		}
		if opts.AllowWildcard {
			rmAction.AllowWildcard = opts.AllowWildcard
		}
	}

	action := &pb.FileAction{
		Action: &pb.FileAction_Rm{
			Rm: rmAction,
		},
	}

	op := NewFileOp([]*pb.FileAction{action})
	return NewFileState(state, op, luaFile, luaLine)
}

func Symlink(state *dag.State, oldpath, newpath string, luaFile string, luaLine int) *dag.State {
	if oldpath == "" || newpath == "" {
		return nil
	}

	symlinkAction := &pb.FileActionSymlink{
		Oldpath: oldpath,
		Newpath: newpath,
	}

	action := &pb.FileAction{
		Action: &pb.FileAction_Symlink{
			Symlink: symlinkAction,
		},
	}

	op := NewFileOp([]*pb.FileAction{action})
	return NewFileState(state, op, luaFile, luaLine)
}

func buildChownOpt(opt *ChownOpt) *pb.ChownOpt {
	if opt == nil {
		return nil
	}

	chown := &pb.ChownOpt{}

	if opt.User != nil {
		if opt.User.Name != "" {
			chown.User = &pb.UserOpt{
				User: &pb.UserOpt_ByName{
					ByName: &pb.NamedUserOpt{
						Name: opt.User.Name,
					},
				},
			}
		} else if opt.User.ID != 0 {
			chown.User = &pb.UserOpt{
				User: &pb.UserOpt_ByID{
					ByID: uint32(opt.User.ID),
				},
			}
		}
	}

	if opt.Group != nil {
		if opt.Group.Name != "" {
			chown.Group = &pb.UserOpt{
				User: &pb.UserOpt_ByName{
					ByName: &pb.NamedUserOpt{
						Name: opt.Group.Name,
					},
				},
			}
		} else if opt.Group.ID != 0 {
			chown.Group = &pb.UserOpt{
				User: &pb.UserOpt_ByID{
					ByID: uint32(opt.Group.ID),
				},
			}
		}
	}

	return chown
}
