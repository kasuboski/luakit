package luavm

import (
	"fmt"
	"maps"
	"strings"
	"unicode"

	pb "github.com/moby/buildkit/solver/pb"
	dockerspec "github.com/moby/docker-image-spec/specs-go/v1"
	lua "github.com/yuin/gopher-lua"

	"github.com/kasuboski/luakit/pkg/dag"
	"github.com/kasuboski/luakit/pkg/ops"
)

var (
	exportedState       *dag.State
	exportedImageConfig *dockerspec.DockerOCIImage
)

func registerAPI(L *lua.LState) {
	bk := L.NewTable()

	L.SetField(bk, "image", L.NewFunction(bkImage))
	L.SetField(bk, "scratch", L.NewFunction(bkScratch))
	L.SetField(bk, "local_", L.NewFunction(bkLocal))
	L.SetField(bk, "git", L.NewFunction(bkGit))
	L.SetField(bk, "http", L.NewFunction(bkHTTP))
	L.SetField(bk, "https", L.NewFunction(bkHTTPS))
	L.SetField(bk, "export", L.NewFunction(bkExport))
	L.SetField(bk, "cache", L.NewFunction(bkCache))
	L.SetField(bk, "secret", L.NewFunction(bkSecret))
	L.SetField(bk, "ssh", L.NewFunction(bkSSH))
	L.SetField(bk, "tmpfs", L.NewFunction(bkTmpfs))
	L.SetField(bk, "bind", L.NewFunction(bkBind))
	L.SetField(bk, "merge", L.NewFunction(bkMerge))
	L.SetField(bk, "diff", L.NewFunction(bkDiff))
	L.SetField(bk, "platform", L.NewFunction(bkPlatform))

	L.SetGlobal("bk", bk)
}

func bkImage(L *lua.LState) int {
	refArg := L.Get(1)
	if refArg.Type() != lua.LTString {
		L.ArgError(1, "string expected")
		return 0
	}
	ref := refArg.String()

	if ref == "" || isWhitespaceOnly(ref) {
		L.RaiseError("bk.image: identifier must not be empty")
		return 0
	}

	var platform *pb.Platform
	if L.GetTop() >= 2 {
		opts := L.CheckTable(2)
		platform = parsePlatform(L, opts)
	}

	file, line := getCallSite(L)
	state := ops.Image(ref, file, line, platform)
	if state == nil {
		L.RaiseError("bk.image: failed to create image state")
		return 0
	}

	L.Push(newState(L, state))
	return 1
}

func isWhitespaceOnly(s string) bool {
	for _, r := range s {
		if !unicode.IsSpace(r) {
			return false
		}
	}
	return true
}

func bkScratch(L *lua.LState) int {
	if L.GetTop() > 0 {
		L.RaiseError("bk.scratch: accepts no arguments")
		return 0
	}

	state := ops.Scratch()
	if state == nil {
		L.RaiseError("bk.scratch: failed to create scratch state")
		return 0
	}

	L.Push(newState(L, state))
	return 1
}

func bkLocal(L *lua.LState) int {
	nameArg := L.Get(1)
	if nameArg.Type() != lua.LTString {
		L.ArgError(1, "string expected")
		return 0
	}
	name := nameArg.String()

	if name == "" || isWhitespaceOnly(name) {
		L.RaiseError("bk.local_: name must not be empty")
		return 0
	}

	file, line := getCallSite(L)

	var opts *ops.LocalOptions
	if L.GetTop() >= 2 {
		optsTable := L.CheckTable(2)
		opts = parseLocalOptions(L, optsTable)
	}

	state := ops.Local(name, file, line, opts)
	if state == nil {
		L.RaiseError("bk.local_: failed to create local state")
		return 0
	}

	L.Push(newState(L, state))
	return 1
}

func bkGit(L *lua.LState) int {
	urlArg := L.Get(1)
	if urlArg.Type() != lua.LTString {
		L.ArgError(1, "string expected")
		return 0
	}
	url := urlArg.String()

	if url == "" || isWhitespaceOnly(url) {
		L.RaiseError("bk.git: URL must not be empty")
		return 0
	}

	file, line := getCallSite(L)

	var opts *ops.GitOptions
	if L.GetTop() >= 2 {
		optsTable := L.CheckTable(2)
		opts = parseGitOptions(L, optsTable)
	}

	state := ops.Git(url, file, line, opts)
	if state == nil {
		L.RaiseError("bk.git: failed to create git state")
		return 0
	}

	L.Push(newState(L, state))
	return 1
}

func bkHTTP(L *lua.LState) int {
	urlArg := L.Get(1)
	if urlArg.Type() != lua.LTString {
		L.ArgError(1, "string expected")
		return 0
	}
	url := urlArg.String()

	if url == "" || isWhitespaceOnly(url) {
		L.RaiseError("bk.http: URL must not be empty")
		return 0
	}

	file, line := getCallSite(L)

	var opts *ops.HTTPOptions
	if L.GetTop() >= 2 {
		optsTable := L.CheckTable(2)
		opts = parseHTTPOptions(L, optsTable)
	}

	state := ops.HTTP(url, file, line, opts)
	if state == nil {
		L.RaiseError("bk.http: failed to create http state")
		return 0
	}

	L.Push(newState(L, state))
	return 1
}

func bkHTTPS(L *lua.LState) int {
	urlArg := L.Get(1)
	if urlArg.Type() != lua.LTString {
		L.ArgError(1, "string expected")
		return 0
	}
	url := urlArg.String()

	if url == "" || isWhitespaceOnly(url) {
		L.RaiseError("bk.https: URL must not be empty")
		return 0
	}

	file, line := getCallSite(L)

	var opts *ops.HTTPOptions
	if L.GetTop() >= 2 {
		optsTable := L.CheckTable(2)
		opts = parseHTTPOptions(L, optsTable)
	}

	state := ops.HTTP(url, file, line, opts)
	if state == nil {
		L.RaiseError("bk.https: failed to create http state")
		return 0
	}

	L.Push(newState(L, state))
	return 1
}

func bkExport(L *lua.LState) int {
	state := checkState(L, 1)

	if exportedState != nil {
		L.RaiseError("bk.export: already called once")
		return 0
	}

	var exportOpts *lua.LTable
	if L.GetTop() >= 2 {
		exportOpts = L.CheckTable(2)
	}

	exportedState = state

	if exportOpts != nil {
		imageConfig := parseExportOptions(L, exportOpts)
		if imageConfig != nil {
			exportedImageConfig = imageConfig
		}
	}

	return 0
}

func parseExportOptions(L *lua.LState, opts *lua.LTable) *dockerspec.DockerOCIImage {
	config := &dockerspec.DockerOCIImage{}
	config.OS = "linux"
	config.Architecture = "amd64"
	config.Config.Env = []string{}
	config.Config.ExposedPorts = make(map[string]struct{})
	config.Config.Labels = make(map[string]string)

	if entrypointVal := L.GetField(opts, "entrypoint"); entrypointVal.Type() == lua.LTTable {
		entrypointTable := entrypointVal.(*lua.LTable)
		config.Config.Entrypoint = luaTableToStringSlice(L, entrypointTable)
	}

	if cmdVal := L.GetField(opts, "cmd"); cmdVal.Type() == lua.LTTable {
		cmdTable := cmdVal.(*lua.LTable)
		config.Config.Cmd = luaTableToStringSlice(L, cmdTable)
	}

	if envVal := L.GetField(opts, "env"); envVal.Type() == lua.LTTable {
		envTable := envVal.(*lua.LTable)
		env := parseEnvTable(L, envTable)
		config.Config.Env = append(config.Config.Env, env...)
	}

	if workdirVal := L.GetField(opts, "workdir"); workdirVal.Type() == lua.LTString {
		config.Config.WorkingDir = workdirVal.String()
	}

	if userVal := L.GetField(opts, "user"); userVal.Type() == lua.LTString {
		config.Config.User = userVal.String()
	}

	if labelsVal := L.GetField(opts, "labels"); labelsVal.Type() == lua.LTTable {
		labelsTable := labelsVal.(*lua.LTable)
		labels := parseLabelsTable(L, labelsTable)
		maps.Copy(config.Config.Labels, labels)
	}

	if exposeVal := L.GetField(opts, "expose"); exposeVal.Type() == lua.LTTable {
		exposeTable := exposeVal.(*lua.LTable)
		ports := luaTableToStringSlice(L, exposeTable)
		for _, port := range ports {
			config.Config.ExposedPorts[port] = struct{}{}
		}
	}

	if osVal := L.GetField(opts, "os"); osVal.Type() == lua.LTString {
		config.OS = osVal.String()
	}

	if archVal := L.GetField(opts, "arch"); archVal.Type() == lua.LTString {
		config.Architecture = archVal.String()
	}

	if variantVal := L.GetField(opts, "variant"); variantVal.Type() == lua.LTString {
		config.Variant = variantVal.String()
	}

	return config
}

func parseLabelsTable(L *lua.LState, table *lua.LTable) map[string]string {
	labels := make(map[string]string)
	table.ForEach(func(key, value lua.LValue) {
		keyStr := key.String()
		valueStr := value.String()
		labels[keyStr] = valueStr
	})
	return labels
}

func GetExportedImageConfig() *dockerspec.DockerOCIImage {
	return exportedImageConfig
}

func parsePlatform(L *lua.LState, opts *lua.LTable) *pb.Platform {
	platformVal := L.GetField(opts, "platform")
	if platformVal.Type() == lua.LTNil {
		return nil
	}

	var platform *pb.Platform
	switch platformVal.Type() {
	case lua.LTString:
		str := platformVal.String()
		platform = parsePlatformString(str)
	case lua.LTTable:
		platform = parsePlatformTable(L, platformVal.(*lua.LTable))
	case lua.LTUserData:
		ud := platformVal.(*lua.LUserData)
		if p, ok := ud.Value.(*pb.Platform); ok {
			platform = p
		}
	}

	return platform
}

func parsePlatformString(str string) *pb.Platform {
	parts := parsePlatformStringParts(str)
	if len(parts) == 0 {
		return nil
	}

	platform := &pb.Platform{}

	for _, part := range parts {
		if part.Key == "os" {
			platform.OS = part.Value
		} else if part.Key == "arch" || part.Key == "architecture" {
			platform.Architecture = part.Value
		} else if part.Key == "variant" {
			platform.Variant = part.Value
		}
	}

	return platform
}

type platformPart struct {
	Key   string
	Value string
}

func parsePlatformStringParts(str string) []platformPart {
	if str == "" {
		return nil
	}

	parts := []platformPart{}

	sep := "/"
	if strings.Contains(str, sep) {
		segments := strings.Split(str, sep)
		if len(segments) >= 2 {
			parts = append(parts, platformPart{Key: "os", Value: segments[0]})
			parts = append(parts, platformPart{Key: "arch", Value: segments[1]})
			if len(segments) >= 3 {
				parts = append(parts, platformPart{Key: "variant", Value: segments[2]})
			}
		}
	} else {
		parts = append(parts, platformPart{Key: "arch", Value: str})
	}

	return parts
}

func parseLocalOptions(L *lua.LState, opts *lua.LTable) *ops.LocalOptions {
	localOpts := &ops.LocalOptions{}

	if includeVal := L.GetField(opts, "include"); includeVal.Type() == lua.LTTable {
		includeTable := includeVal.(*lua.LTable)
		localOpts.IncludePatterns = luaTableToStringSlice(L, includeTable)
	}

	if excludeVal := L.GetField(opts, "exclude"); excludeVal.Type() == lua.LTTable {
		excludeTable := excludeVal.(*lua.LTable)
		localOpts.ExcludePatterns = luaTableToStringSlice(L, excludeTable)
	}

	if sharedKeyHintVal := L.GetField(opts, "shared_key_hint"); sharedKeyHintVal.Type() == lua.LTString {
		localOpts.SharedKeyHint = sharedKeyHintVal.String()
	}

	return localOpts
}

func parseGitOptions(L *lua.LState, opts *lua.LTable) *ops.GitOptions {
	gitOpts := &ops.GitOptions{}

	if refVal := L.GetField(opts, "ref"); refVal.Type() == lua.LTString {
		gitOpts.Ref = refVal.String()
	}

	if keepGitDirVal := L.GetField(opts, "keep_git_dir"); keepGitDirVal.Type() == lua.LTBool {
		gitOpts.KeepGitDir = bool(keepGitDirVal.(lua.LBool))
	}

	return gitOpts
}

func parseHTTPOptions(L *lua.LState, opts *lua.LTable) *ops.HTTPOptions {
	httpOpts := &ops.HTTPOptions{}

	if checksumVal := L.GetField(opts, "checksum"); checksumVal.Type() == lua.LTString {
		httpOpts.Checksum = checksumVal.String()
	}

	if filenameVal := L.GetField(opts, "filename"); filenameVal.Type() == lua.LTString {
		httpOpts.Filename = filenameVal.String()
	}

	if modeVal := L.GetField(opts, "chmod"); modeVal.Type() == lua.LTNumber {
		httpOpts.Mode = int32(modeVal.(lua.LNumber))
	}

	if headersVal := L.GetField(opts, "headers"); headersVal.Type() == lua.LTTable {
		headersTable := headersVal.(*lua.LTable)
		httpOpts.Headers = make(map[string]string)
		headersTable.ForEach(func(key, value lua.LValue) {
			keyStr := key.String()
			valueStr := value.String()
			httpOpts.Headers[keyStr] = valueStr
		})
	}

	if usernameVal := L.GetField(opts, "username"); usernameVal.Type() == lua.LTString {
		httpOpts.Username = usernameVal.String()
	}

	if passwordVal := L.GetField(opts, "password"); passwordVal.Type() == lua.LTString {
		httpOpts.Password = passwordVal.String()
	}

	return httpOpts
}

func parsePlatformTable(L *lua.LState, table *lua.LTable) *pb.Platform {
	platform := &pb.Platform{}

	if osVal := L.GetField(table, "os"); osVal.Type() == lua.LTString {
		platform.OS = osVal.String()
	}
	if archVal := L.GetField(table, "arch"); archVal.Type() == lua.LTString {
		platform.Architecture = archVal.String()
	}
	if variantVal := L.GetField(table, "variant"); variantVal.Type() == lua.LTString {
		platform.Variant = variantVal.String()
	}

	return platform
}

func bkPlatform(L *lua.LState) int {
	nArgs := L.GetTop()
	if nArgs == 0 {
		L.RaiseError("bk.platform: requires at least one argument (os and arch, or a platform string)")
		return 0
	}

	platform := &pb.Platform{}

	if nArgs == 1 {
		arg := L.Get(1)
		if arg.Type() != lua.LTString {
			L.ArgError(1, "string expected")
			return 0
		}
		platformStr := arg.String()

		parts := parsePlatformStringParts(platformStr)
		if len(parts) == 0 {
			L.RaiseError("bk.platform: invalid platform string '%s'", platformStr)
			return 0
		}

		for _, part := range parts {
			if part.Key == "os" {
				platform.OS = part.Value
			} else if part.Key == "arch" || part.Key == "architecture" {
				platform.Architecture = part.Value
			} else if part.Key == "variant" {
				platform.Variant = part.Value
			}
		}
	} else {
		osArg := L.Get(1)
		if osArg.Type() != lua.LTString {
			L.ArgError(1, "string expected")
			return 0
		}
		platform.OS = osArg.String()

		if nArgs >= 2 {
			archArg := L.Get(2)
			if archArg.Type() != lua.LTString {
				L.ArgError(2, "string expected")
				return 0
			}
			platform.Architecture = archArg.String()
		}

		if nArgs >= 3 {
			variantArg := L.Get(3)
			if variantArg.Type() != lua.LTString {
				L.ArgError(3, "string expected")
				return 0
			}
			platform.Variant = variantArg.String()
		}
	}

	ud := L.NewUserData()
	ud.Value = platform
	L.SetMetatable(ud, L.GetTypeMetatable(luaPlatformTypeName))

	L.Push(ud)
	return 1
}

func platformToString(L *lua.LState) int {
	ud := L.CheckUserData(1)
	platform, ok := ud.Value.(*pb.Platform)
	if !ok {
		L.RaiseError("bk.platform: expected platform userdata")
		return 0
	}

	result := platform.OS + "/" + platform.Architecture
	if platform.Variant != "" {
		result += "/" + platform.Variant
	}

	L.Push(lua.LString(result))
	return 1
}

func getCallSite(L *lua.LState) (string, int) {
	info, ok := L.GetStack(1)
	if !ok {
		return "", 0
	}

	file, line := getLuaSourceLocation(L, info)
	return file, line
}

const (
	luaMountTypeName    = "luakit.mount"
	luaPlatformTypeName = "luakit.platform"
)

func registerMountType(L *lua.LState) {
	mt := L.NewTypeMetatable(luaMountTypeName)
	L.SetGlobal(luaMountTypeName, mt)

	L.SetField(mt, "__tostring", L.NewFunction(mountToString))
}

func registerPlatformType(L *lua.LState) {
	mt := L.NewTypeMetatable(luaPlatformTypeName)
	L.SetGlobal(luaPlatformTypeName, mt)

	L.SetField(mt, "__tostring", L.NewFunction(platformToString))
}

func newMount(L *lua.LState, mount *ops.Mount) *lua.LUserData {
	ud := L.NewUserData()
	ud.Value = mount
	L.SetMetatable(ud, L.GetTypeMetatable(luaMountTypeName))
	return ud
}

func checkMount(L *lua.LState, n int) *ops.Mount {
	ud := L.CheckUserData(n)
	if v, ok := ud.Value.(*ops.Mount); ok {
		return v
	}
	L.ArgError(n, fmt.Sprintf("expected %s, got %s", luaMountTypeName, ud.Type().String()))
	return nil
}

func mountToString(L *lua.LState) int {
	mount := checkMount(L, 1)
	L.Push(lua.LString(fmt.Sprintf("luakit.mount@%p", mount)))
	return 1
}

func bkCache(L *lua.LState) int {
	if L.GetTop() < 1 {
		L.RaiseError("bk.cache: destination argument required")
		return 0
	}

	destArg := L.Get(1)
	if destArg.Type() != lua.LTString {
		L.ArgError(1, "string expected")
		return 0
	}
	dest := destArg.String()

	var opts *ops.CacheOptions
	if L.GetTop() >= 2 {
		optsTable := L.CheckTable(2)
		opts = parseCacheOptions(L, optsTable)
	}

	mount := ops.CacheMount(dest, opts)
	if mount == nil {
		L.RaiseError("bk.cache: failed to create cache mount")
		return 0
	}

	L.Push(newMount(L, mount))
	return 1
}

func parseCacheOptions(L *lua.LState, opts *lua.LTable) *ops.CacheOptions {
	cacheOpts := &ops.CacheOptions{}

	if idVal := L.GetField(opts, "id"); idVal.Type() == lua.LTString {
		cacheOpts.ID = idVal.String()
	}

	if sharingVal := L.GetField(opts, "sharing"); sharingVal.Type() == lua.LTString {
		cacheOpts.Sharing = sharingVal.String()
	}

	return cacheOpts
}

func bkSecret(L *lua.LState) int {
	if L.GetTop() < 1 {
		L.RaiseError("bk.secret: destination argument required")
		return 0
	}

	destArg := L.Get(1)
	if destArg.Type() != lua.LTString {
		L.ArgError(1, "string expected")
		return 0
	}
	dest := destArg.String()

	var opts *ops.SecretOptions
	if L.GetTop() >= 2 {
		optsTable := L.CheckTable(2)
		opts = parseSecretOptions(L, optsTable)
	}

	mount := ops.SecretMount(dest, opts)
	if mount == nil {
		L.RaiseError("bk.secret: failed to create secret mount")
		return 0
	}

	L.Push(newMount(L, mount))
	return 1
}

func parseSecretOptions(L *lua.LState, opts *lua.LTable) *ops.SecretOptions {
	secretOpts := &ops.SecretOptions{}

	if idVal := L.GetField(opts, "id"); idVal.Type() == lua.LTString {
		secretOpts.ID = idVal.String()
	}

	if uidVal := L.GetField(opts, "uid"); uidVal.Type() == lua.LTNumber {
		secretOpts.UID = uint32(uidVal.(lua.LNumber))
	}

	if gidVal := L.GetField(opts, "gid"); gidVal.Type() == lua.LTNumber {
		secretOpts.GID = uint32(gidVal.(lua.LNumber))
	}

	if modeVal := L.GetField(opts, "mode"); modeVal.Type() == lua.LTNumber {
		secretOpts.Mode = uint32(modeVal.(lua.LNumber))
	}

	if optionalVal := L.GetField(opts, "optional"); optionalVal.Type() == lua.LTBool {
		secretOpts.Optional = bool(optionalVal.(lua.LBool))
	}

	return secretOpts
}

func bkSSH(L *lua.LState) int {
	var opts *ops.SSHOptions
	if L.GetTop() >= 1 {
		optsTable := L.CheckTable(1)
		opts = parseSSHOptions(L, optsTable)
	}

	mount := ops.SSHMount(opts)
	if mount == nil {
		L.RaiseError("bk.ssh: failed to create ssh mount")
		return 0
	}

	L.Push(newMount(L, mount))
	return 1
}

func parseSSHOptions(L *lua.LState, opts *lua.LTable) *ops.SSHOptions {
	sshOpts := &ops.SSHOptions{}

	if destVal := L.GetField(opts, "dest"); destVal.Type() == lua.LTString {
		sshOpts.Dest = destVal.String()
	}

	if idVal := L.GetField(opts, "id"); idVal.Type() == lua.LTString {
		sshOpts.ID = idVal.String()
	}

	if uidVal := L.GetField(opts, "uid"); uidVal.Type() == lua.LTNumber {
		sshOpts.UID = uint32(uidVal.(lua.LNumber))
	}

	if gidVal := L.GetField(opts, "gid"); gidVal.Type() == lua.LTNumber {
		sshOpts.GID = uint32(gidVal.(lua.LNumber))
	}

	if modeVal := L.GetField(opts, "mode"); modeVal.Type() == lua.LTNumber {
		sshOpts.Mode = uint32(modeVal.(lua.LNumber))
	}

	if optionalVal := L.GetField(opts, "optional"); optionalVal.Type() == lua.LTBool {
		sshOpts.Optional = bool(optionalVal.(lua.LBool))
	}

	return sshOpts
}

func bkTmpfs(L *lua.LState) int {
	if L.GetTop() < 1 {
		L.RaiseError("bk.tmpfs: destination argument required")
		return 0
	}

	destArg := L.Get(1)
	if destArg.Type() != lua.LTString {
		L.ArgError(1, "string expected")
		return 0
	}
	dest := destArg.String()

	var opts *ops.TmpfsOptions
	if L.GetTop() >= 2 {
		optsTable := L.CheckTable(2)
		opts = parseTmpfsOptions(L, optsTable)
	}

	mount := ops.TmpfsMount(dest, opts)
	if mount == nil {
		L.RaiseError("bk.tmpfs: failed to create tmpfs mount")
		return 0
	}

	L.Push(newMount(L, mount))
	return 1
}

func parseTmpfsOptions(L *lua.LState, opts *lua.LTable) *ops.TmpfsOptions {
	tmpfsOpts := &ops.TmpfsOptions{}

	if sizeVal := L.GetField(opts, "size"); sizeVal.Type() == lua.LTNumber {
		tmpfsOpts.Size = int64(sizeVal.(lua.LNumber))
	}

	return tmpfsOpts
}

func bkBind(L *lua.LState) int {
	if L.GetTop() < 2 {
		L.RaiseError("bk.bind: state and destination arguments required")
		return 0
	}

	state := checkState(L, 1)
	dest := L.CheckString(2)

	var opts *ops.BindOptions
	if L.GetTop() >= 3 {
		optsTable := L.CheckTable(3)
		opts = parseBindOptions(L, optsTable)
	}

	mount := ops.BindMount(state, dest, opts)
	if mount == nil {
		L.RaiseError("bk.bind: failed to create bind mount")
		return 0
	}

	L.Push(newMount(L, mount))
	return 1
}

func parseBindOptions(L *lua.LState, opts *lua.LTable) *ops.BindOptions {
	bindOpts := &ops.BindOptions{
		Readonly: true,
	}

	if selectorVal := L.GetField(opts, "selector"); selectorVal.Type() == lua.LTString {
		bindOpts.Selector = selectorVal.String()
	}

	if readonlyVal := L.GetField(opts, "readonly"); readonlyVal.Type() == lua.LTBool {
		bindOpts.Readonly = bool(readonlyVal.(lua.LBool))
	}

	return bindOpts
}

func bkMerge(L *lua.LState) int {
	if L.GetTop() < 2 {
		L.RaiseError("bk.merge: requires at least 2 states")
		return 0
	}

	states := []*dag.State{}

	for i := 1; i <= L.GetTop(); i++ {
		state := checkState(L, i)
		states = append(states, state)
	}

	file, line := getCallSite(L)
	result := ops.Merge(states, file, line)
	if result == nil {
		L.RaiseError("bk.merge: failed to create merge state (requires at least 2 states)")
		return 0
	}

	L.Push(newState(L, result))
	return 1
}

func bkDiff(L *lua.LState) int {
	if L.GetTop() < 2 {
		L.RaiseError("bk.diff: requires lower and upper state arguments")
		return 0
	}

	lowerState := checkState(L, 1)
	upperState := checkState(L, 2)

	file, line := getCallSite(L)
	result := ops.Diff(lowerState, upperState, file, line)
	if result == nil {
		L.RaiseError("bk.diff: failed to create diff state")
		return 0
	}

	L.Push(newState(L, result))
	return 1
}
