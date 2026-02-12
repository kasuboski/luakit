package luavm

import (
	"fmt"

	pb "github.com/moby/buildkit/solver/pb"
	lua "github.com/yuin/gopher-lua"

	"github.com/kasuboski/luakit/pkg/dag"
	"github.com/kasuboski/luakit/pkg/ops"
)

const (
	luaStateTypeName = "luakit.state"
)

func registerStateType(L *lua.LState) {
	mt := L.NewTypeMetatable(luaStateTypeName)
	L.SetGlobal(luaStateTypeName, mt)

	L.SetField(mt, "__index", L.NewFunction(stateIndex))
	L.SetField(mt, "__tostring", L.NewFunction(stateToString))
}

func newState(L *lua.LState, state *dag.State) *lua.LUserData {
	ud := L.NewUserData()
	ud.Value = state
	L.SetMetatable(ud, L.GetTypeMetatable(luaStateTypeName))
	return ud
}

func checkState(L *lua.LState, n int) *dag.State {
	ud := L.CheckUserData(n)
	if v, ok := ud.Value.(*dag.State); ok {
		return v
	}
	L.ArgError(n, fmt.Sprintf("expected %s, got %s", luaStateTypeName, ud.Type().String()))
	return nil
}

func stateIndex(L *lua.LState) int {
	checkState(L, 1)
	key := L.CheckString(2)

	switch key {
	case "run":
		L.Push(L.NewClosure(stateRun, L.NewFunction(stateRun)))
		return 1
	case "copy":
		L.Push(L.NewClosure(stateCopy, L.NewFunction(stateCopy)))
		return 1
	case "mkdir":
		L.Push(L.NewClosure(stateMkdir, L.NewFunction(stateMkdir)))
		return 1
	case "mkfile":
		L.Push(L.NewClosure(stateMkfile, L.NewFunction(stateMkfile)))
		return 1
	case "rm":
		L.Push(L.NewClosure(stateRm, L.NewFunction(stateRm)))
		return 1
	case "symlink":
		L.Push(L.NewClosure(stateSymlink, L.NewFunction(stateSymlink)))
		return 1
	case "with_metadata":
		L.Push(L.NewClosure(stateWithMetadata, L.NewFunction(stateWithMetadata)))
		return 1
	default:
		L.RaiseError("unknown field: %s", key)
		return 0
	}
}

func stateToString(L *lua.LState) int {
	state := checkState(L, 1)
	L.Push(lua.LString(fmt.Sprintf("luakit.state@%p", state)))
	return 1
}

func stateRun(L *lua.LState) int {
	state := checkState(L, 1)

	if L.GetTop() < 2 {
		L.RaiseError("run: command argument required")
		return 0
	}

	var cmd []string
	cmdArg := L.CheckAny(2)

	switch cmdArg.Type() {
	case lua.LTString:
		cmdStr := cmdArg.String()
		cmd = []string{"/bin/sh", "-c", cmdStr}
	case lua.LTTable:
		table := cmdArg.(*lua.LTable)
		cmd = luaTableToStringSlice(L, table)
	default:
		L.ArgError(2, "command must be string or table")
		return 0
	}

	var opts *ops.ExecOptions
	if L.GetTop() >= 3 {
		optsArg := L.Get(3)
		if optsArg.Type() != lua.LTNil {
			optsTable := L.CheckTable(3)
			opts = parseExecOptions(L, optsTable)
		}
	}

	file, line := getCallSite(L)

	result := ops.Run(state, cmd, opts, file, line)
	if result == nil {
		L.RaiseError("run: failed to create exec state")
		return 0
	}

	L.Push(newState(L, result))
	return 1
}

func parseExecOptions(L *lua.LState, opts *lua.LTable) *ops.ExecOptions {
	execOpts := &ops.ExecOptions{}

	envVal := L.GetField(opts, "env")
	if envVal.Type() == lua.LTTable {
		envTable := envVal.(*lua.LTable)
		execOpts.Env = parseEnvTable(L, envTable)
	} else if envVal.Type() != lua.LTNil {
		L.RaiseError("run options env must be a table")
		return nil
	}

	cwdVal := L.GetField(opts, "cwd")
	if cwdVal.Type() == lua.LTString {
		execOpts.Cwd = cwdVal.String()
	} else if cwdVal.Type() != lua.LTNil {
		L.RaiseError("run options cwd must be a string")
		return nil
	}

	userVal := L.GetField(opts, "user")
	if userVal.Type() == lua.LTString {
		execOpts.User = userVal.String()
	} else if userVal.Type() != lua.LTNil {
		L.RaiseError("run options user must be a string")
		return nil
	}

	networkVal := L.GetField(opts, "network")
	if networkVal.Type() == lua.LTString {
		network := networkVal.String()
		execOpts.Network = &network
	} else if networkVal.Type() != lua.LTNil {
		L.RaiseError("run options network must be a string")
		return nil
	}

	securityVal := L.GetField(opts, "security")
	if securityVal.Type() == lua.LTString {
		security := securityVal.String()
		execOpts.Security = &security
	} else if securityVal.Type() != lua.LTNil {
		L.RaiseError("run options security must be a string")
		return nil
	}

	mountsVal := L.GetField(opts, "mounts")
	if mountsVal.Type() == lua.LTTable {
		mountsTable := mountsVal.(*lua.LTable)
		execOpts.Mounts = parseMountsTable(L, mountsTable)
	} else if mountsVal.Type() != lua.LTNil {
		L.RaiseError("run options mounts must be a table")
		return nil
	}

	hostnameVal := L.GetField(opts, "hostname")
	if hostnameVal.Type() == lua.LTString {
		execOpts.Hostname = hostnameVal.String()
	} else if hostnameVal.Type() != lua.LTNil {
		L.RaiseError("run options hostname must be a string")
		return nil
	}

	validExitCodesVal := L.GetField(opts, "valid_exit_codes")
	if validExitCodesVal.Type() == lua.LTTable {
		validExitCodesTable := validExitCodesVal.(*lua.LTable)
		execOpts.ValidExitCodes = parseValidExitCodes(L, validExitCodesTable)
	} else if validExitCodesVal.Type() == lua.LTNumber {
		code := int32(validExitCodesVal.(lua.LNumber))
		execOpts.ValidExitCodes = []int32{code}
	} else if validExitCodesVal.Type() == lua.LTString {
		rangeStr := validExitCodesVal.String()
		codes, err := parseExitCodeRange(rangeStr)
		if err != nil {
			L.RaiseError("run options valid_exit_codes: %v", err)
			return nil
		}
		execOpts.ValidExitCodes = codes
	} else if validExitCodesVal.Type() != lua.LTNil {
		L.RaiseError("run options valid_exit_codes must be a number, string, or table")
		return nil
	}

	return execOpts
}

func parseMountsTable(L *lua.LState, table *lua.LTable) []*ops.Mount {
	var mounts []*ops.Mount
	for i := int64(1); ; i++ {
		val := L.RawGet(table, lua.LNumber(i))
		if val.Type() == lua.LTNil {
			break
		}
		if mount := checkMountOrNil(L, val); mount != nil {
			mounts = append(mounts, mount)
		}
	}
	return mounts
}

func checkMountOrNil(L *lua.LState, val lua.LValue) *ops.Mount {
	if val.Type() != lua.LTUserData {
		return nil
	}
	ud := val.(*lua.LUserData)
	if mount, ok := ud.Value.(*ops.Mount); ok {
		return mount
	}
	return nil
}

func parseEnvTable(L *lua.LState, table *lua.LTable) []string {
	var env []string
	table.ForEach(func(key, value lua.LValue) {
		keyStr := key.String()
		valueStr := value.String()
		env = append(env, keyStr+"="+valueStr)
	})
	return env
}

func parseValidExitCodes(L *lua.LState, table *lua.LTable) []int32 {
	var codes []int32
	for i := int64(1); ; i++ {
		val := L.RawGet(table, lua.LNumber(i))
		if val.Type() == lua.LTNil {
			break
		}
		if val.Type() == lua.LTNumber {
			codes = append(codes, int32(val.(lua.LNumber)))
		}
	}
	return codes
}

func parseExitCodeRange(s string) ([]int32, error) {
	var start, end int32
	n, err := fmt.Sscanf(s, "%d..%d", &start, &end)
	if err != nil || n != 2 {
		return nil, fmt.Errorf("invalid range format, expected 'start..end' (e.g., '0..5')")
	}
	if start > end {
		return nil, fmt.Errorf("invalid range: start (%d) must be <= end (%d)", start, end)
	}
	if start < 0 || end > 255 {
		return nil, fmt.Errorf("invalid range: exit codes must be between 0 and 255")
	}

	codes := make([]int32, end-start+1)
	for i := start; i <= end; i++ {
		codes[i-start] = i
	}
	return codes, nil
}

func luaTableToStringSlice(L *lua.LState, table *lua.LTable) []string {
	var result []string
	for i := int64(1); ; i++ {
		val := L.RawGet(table, lua.LNumber(i))
		if val.Type() == lua.LTNil {
			break
		}
		result = append(result, val.String())
	}
	return result
}

func getLuaSourceLocation(L *lua.LState, info *lua.Debug) (string, int) {
	_, err := L.GetInfo("Sl", info, nil)
	if err != nil {
		return "", 0
	}

	file := info.Source
	line := info.CurrentLine

	return file, line
}

func stateCopy(L *lua.LState) int {
	state := checkState(L, 1)

	if L.GetTop() < 4 {
		L.RaiseError("copy: requires from, src, and dest arguments")
		return 0
	}

	fromState := checkState(L, 2)
	src := L.CheckString(3)
	dest := L.CheckString(4)

	var opts *ops.CopyOptions
	if L.GetTop() >= 5 {
		optsTable := L.CheckTable(5)
		var err error
		opts, err = parseCopyOptions(L, optsTable)
		if err != nil {
			L.RaiseError("copy: %v", err)
			return 0
		}
	}

	file, line := getCallSite(L)

	result := ops.Copy(state, fromState, src, dest, opts, file, line)
	if result == nil {
		L.RaiseError("copy: failed to create file state")
		return 0
	}

	L.Push(newState(L, result))
	return 1
}

func stateMkdir(L *lua.LState) int {
	state := checkState(L, 1)

	if L.GetTop() < 2 {
		L.RaiseError("mkdir: path argument required")
		return 0
	}

	path := L.CheckString(2)
	if path == "" {
		L.RaiseError("mkdir: path must not be empty")
		return 0
	}

	var opts *ops.MkdirOptions
	if L.GetTop() >= 3 {
		optsTable := L.CheckTable(3)
		var err error
		opts, err = parseMkdirOptions(L, optsTable)
		if err != nil {
			L.RaiseError("mkdir: %v", err)
			return 0
		}
	}

	file, line := getCallSite(L)

	result := ops.Mkdir(state, path, opts, file, line)
	if result == nil {
		L.RaiseError("mkdir: failed to create file state")
		return 0
	}

	L.Push(newState(L, result))
	return 1
}

func stateMkfile(L *lua.LState) int {
	state := checkState(L, 1)

	if L.GetTop() < 3 {
		L.RaiseError("mkfile: path and data arguments required")
		return 0
	}

	path := L.CheckString(2)
	data := L.CheckString(3)

	var opts *ops.MkfileOptions
	if L.GetTop() >= 4 {
		optsTable := L.CheckTable(4)
		var err error
		opts, err = parseMkfileOptions(L, optsTable)
		if err != nil {
			L.RaiseError("mkfile: %v", err)
			return 0
		}
	}

	file, line := getCallSite(L)

	result := ops.Mkfile(state, path, data, opts, file, line)
	if result == nil {
		L.RaiseError("mkfile: failed to create file state")
		return 0
	}

	L.Push(newState(L, result))
	return 1
}

func stateRm(L *lua.LState) int {
	state := checkState(L, 1)

	if L.GetTop() < 2 {
		L.RaiseError("rm: path argument required")
		return 0
	}

	path := L.CheckString(2)
	if path == "" {
		L.RaiseError("rm: path must not be empty")
		return 0
	}

	var opts *ops.RmOptions
	if L.GetTop() >= 3 {
		optsTable := L.CheckTable(3)
		opts = parseRmOptions(L, optsTable)
	}

	file, line := getCallSite(L)

	result := ops.Rm(state, path, opts, file, line)
	if result == nil {
		L.RaiseError("rm: failed to create file state")
		return 0
	}

	L.Push(newState(L, result))
	return 1
}

func stateSymlink(L *lua.LState) int {
	state := checkState(L, 1)

	if L.GetTop() < 3 {
		L.RaiseError("symlink: oldpath and newpath arguments required")
		return 0
	}

	oldpath := L.CheckString(2)
	newpath := L.CheckString(3)

	file, line := getCallSite(L)

	result := ops.Symlink(state, oldpath, newpath, file, line)
	if result == nil {
		L.RaiseError("symlink: failed to create file state")
		return 0
	}

	L.Push(newState(L, result))
	return 1
}

func stateWithMetadata(L *lua.LState) int {
	state := checkState(L, 1)

	if L.GetTop() < 2 {
		L.RaiseError("with_metadata: options argument required")
		return 0
	}

	optsTable := L.CheckTable(2)
	opts := parseMetadataOptions(L, optsTable)

	if opts == nil {
		L.RaiseError("with_metadata: failed to parse metadata options")
		return 0
	}

	result := ops.WithMetadata(state, opts)
	if result == nil {
		L.RaiseError("with_metadata: failed to apply metadata")
		return 0
	}

	L.Push(newState(L, result))
	return 1
}

func parseMetadataOptions(L *lua.LState, opts *lua.LTable) *pb.OpMetadata {
	meta := &pb.OpMetadata{}

	if descriptionVal := L.GetField(opts, "description"); descriptionVal.Type() == lua.LTString {
		if meta.Description == nil {
			meta.Description = make(map[string]string, 1)
		}
		meta.Description["llb.custom"] = descriptionVal.String()
	}

	if progressGroupVal := L.GetField(opts, "progress_group"); progressGroupVal.Type() == lua.LTString {
		meta.ProgressGroup = &pb.ProgressGroup{
			Id: progressGroupVal.String(),
		}
	}

	return meta
}

func parseCopyOptions(L *lua.LState, opts *lua.LTable) (*ops.CopyOptions, error) {
	copyOpts := &ops.CopyOptions{}

	if modeVal := L.GetField(opts, "mode"); modeVal.Type() == lua.LTString {
		modeStr := modeVal.String()
		var mode int32
		n, err := fmt.Sscanf(modeStr, "%o", &mode)
		if err != nil || n != 1 {
			return nil, fmt.Errorf("invalid mode string: %s", modeStr)
		}
		copyOpts.Mode = mode
	} else if modeVal.Type() == lua.LTNumber {
		// In Lua, numbers like 0755 are just decimal 755
		// But for modes, we want to interpret them as octal
		// If the number looks like it's meant to be octal (e.g., 0755),
		// we should convert it. However, Lua doesn't preserve the leading 0,
		// so we can't tell if 755 was meant to be 0755 or 0755 decimal.
		// For now, we'll just use the number as-is.
		// Users should use string mode (e.g., "0755") for octal values.
		copyOpts.Mode = int32(modeVal.(lua.LNumber))
	}

	if followSymlinkVal := L.GetField(opts, "follow_symlink"); followSymlinkVal.Type() == lua.LTBool {
		copyOpts.FollowSymlink = bool(followSymlinkVal.(lua.LBool))
	}

	if createDestPathVal := L.GetField(opts, "create_dest_path"); createDestPathVal.Type() == lua.LTBool {
		copyOpts.CreateDestPath = bool(createDestPathVal.(lua.LBool))
	}

	if allowWildcardVal := L.GetField(opts, "allow_wildcard"); allowWildcardVal.Type() == lua.LTBool {
		copyOpts.AllowWildcard = bool(allowWildcardVal.(lua.LBool))
	}

	if includeVal := L.GetField(opts, "include"); includeVal.Type() == lua.LTTable {
		includeTable := includeVal.(*lua.LTable)
		copyOpts.IncludePatterns = luaTableToStringSlice(L, includeTable)
	}

	if excludeVal := L.GetField(opts, "exclude"); excludeVal.Type() == lua.LTTable {
		excludeTable := excludeVal.(*lua.LTable)
		copyOpts.ExcludePatterns = luaTableToStringSlice(L, excludeTable)
	}

	if ownerVal := L.GetField(opts, "owner"); ownerVal.Type() == lua.LTTable {
		ownerTable := ownerVal.(*lua.LTable)
		copyOpts.Owner = parseChownOpt(L, ownerTable)
	}

	return copyOpts, nil
}

func parseMkdirOptions(L *lua.LState, opts *lua.LTable) (*ops.MkdirOptions, error) {
	mkdirOpts := &ops.MkdirOptions{}

	if modeVal := L.GetField(opts, "mode"); modeVal.Type() == lua.LTString {
		modeStr := modeVal.String()
		var mode int32
		n, err := fmt.Sscanf(modeStr, "%o", &mode)
		if err != nil || n != 1 {
			return nil, fmt.Errorf("invalid mode string: %s", modeStr)
		}
		mkdirOpts.Mode = mode
	} else if modeVal.Type() == lua.LTNumber {
		mkdirOpts.Mode = int32(modeVal.(lua.LNumber))
	}

	if makeParentsVal := L.GetField(opts, "make_parents"); makeParentsVal.Type() == lua.LTBool {
		mkdirOpts.MakeParents = bool(makeParentsVal.(lua.LBool))
	}

	if ownerVal := L.GetField(opts, "owner"); ownerVal.Type() == lua.LTTable {
		ownerTable := ownerVal.(*lua.LTable)
		mkdirOpts.Owner = parseChownOpt(L, ownerTable)
	}

	return mkdirOpts, nil
}

func parseMkfileOptions(L *lua.LState, opts *lua.LTable) (*ops.MkfileOptions, error) {
	mkfileOpts := &ops.MkfileOptions{}

	if modeVal := L.GetField(opts, "mode"); modeVal.Type() == lua.LTString {
		modeStr := modeVal.String()
		var mode int32
		n, err := fmt.Sscanf(modeStr, "%o", &mode)
		if err != nil || n != 1 {
			return nil, fmt.Errorf("invalid mode string: %s", modeStr)
		}
		mkfileOpts.Mode = mode
	} else if modeVal.Type() == lua.LTNumber {
		mkfileOpts.Mode = int32(modeVal.(lua.LNumber))
	}

	if ownerVal := L.GetField(opts, "owner"); ownerVal.Type() == lua.LTTable {
		ownerTable := ownerVal.(*lua.LTable)
		mkfileOpts.Owner = parseChownOpt(L, ownerTable)
	}

	return mkfileOpts, nil
}

func parseRmOptions(L *lua.LState, opts *lua.LTable) *ops.RmOptions {
	rmOpts := &ops.RmOptions{}

	if allowNotFoundVal := L.GetField(opts, "allow_not_found"); allowNotFoundVal.Type() == lua.LTBool {
		rmOpts.AllowNotFound = bool(allowNotFoundVal.(lua.LBool))
	}

	if allowWildcardVal := L.GetField(opts, "allow_wildcard"); allowWildcardVal.Type() == lua.LTBool {
		rmOpts.AllowWildcard = bool(allowWildcardVal.(lua.LBool))
	}

	return rmOpts
}

func parseChownOpt(L *lua.LState, opts *lua.LTable) *ops.ChownOpt {
	chown := &ops.ChownOpt{}

	if userVal := L.GetField(opts, "user"); userVal.Type() == lua.LTString {
		chown.User = &ops.UserOpt{
			Name: userVal.String(),
		}
	} else if userVal.Type() == lua.LTNumber {
		chown.User = &ops.UserOpt{
			ID: int64(userVal.(lua.LNumber)),
		}
	}

	if groupVal := L.GetField(opts, "group"); groupVal.Type() == lua.LTString {
		chown.Group = &ops.UserOpt{
			Name: groupVal.String(),
		}
	} else if groupVal.Type() == lua.LTNumber {
		chown.Group = &ops.UserOpt{
			ID: int64(groupVal.(lua.LNumber)),
		}
	}

	return chown
}
